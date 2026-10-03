# Pomiary Oentike

Pomiar: 2026-09-24, Apple M4, macOS/arm64. Wyniki mikrobenchmarków to mediany z trzech przebiegów `go test -bench . -benchmem -count=3`, bez równoległego fuzzingu i kompilacji testów. Nie są pomiarem całej aplikacji ani wynikiem urządzenia mobilnego.

## Powtarzanie

```sh
./oentike bench
./oentike check
cd oentike-web
npm run perf:assets
```

W API można dodatkowo uruchomić:

```sh
go test -race ./...
go test ./internal/spatial -run '^$' -fuzz FuzzDecodeCells -fuzztime=10s
```

## Przestrzeń

Fixture: 429 lub 4290 rozłącznych, syntetycznych wielokątów po 128 wierzchołków. Testy nie wymagają sieci, eksportu BDL ani PostGIS. Wyszukiwanie nadal liniowo skanuje bounding boxy. „Box-only” trafia w prostokąt obwiedni, ale poza właściwy wielokąt.

| Operacja | 429 obszarów | 4290 obszarów | Alokacje zapytania |
|---|---:|---:|---:|
| Trafienie w pierwszy obszar | 238,0 ns | 238,1 ns | 0 |
| Trafienie w ostatni obszar | 543,5 ns | 2982 ns | 0 |
| Punkt poza wszystkimi obwiedniami | 167,4 ns | 2124 ns | 0 |
| Box-only | 435,5 ns | 2629 ns | 0 |
| Otwarcie, walidacja, zamknięcie | 120 976 ns | 1 108 101 ns | 435 / 4296 przy otwieraniu |

Otwarcie mierzone jest z ciepłym cache systemu plików; nie przedstawia opóźnienia dysku. Przy otwarciu powstaje indeks i kopie identyfikatorów (około 40 / 380 kB alokacji Go). Wierzchołki pozostają w mmap; licznik alokacji Go nie obejmuje mapowanych stron ani całkowitego RSS.

Wstępny pomiar starego czytnika na tym samym typie fixture: pierwsze trafienie około 87,8 ns, ostatnie z 429 około 368,3 ns, 16 B i 1 alokacja na trafienie. Nowy czytnik **nie jest ogólnie szybszy**: dekodowanie little-endian bez `unsafe` i jawna obsługa granicy zwiększają koszt małych zapytań. Zyskiem są walidacja przed użyciem danych, brak zależności od natywnego layoutu struktur i brak alokacji podczas wyszukiwania. Priorytetem tej zmiany jest poprawność formatu, nie rekord przepustowości.

Nie są sprawdzane: pełna poprawność topologiczna pierścieni, otwory ani wielopoligony. Obecny format nie przechowuje dwóch ostatnich. Otwarty plik musi pozostać niezmienny; aktualizacja przez nowy plik i atomową podmianę, nie zapis w miejscu. `Close` nie może wyścigać się z zapytaniami.

## Scoring

| Operacja | Czas | Pamięć Go | Alokacje |
|---|---:|---:|---:|
| Scoring V1 | 2056 ns | 2154 B | 42 |
| Scoring V2 | 3673 ns | 3811 B | 75 |

Pomiar obejmuje liczenie wyniku, serializację czynników i hash wejścia. Nie obejmuje pobierania pogody, SQL, gRPC ani interfejsu. Na tej podstawie nie ma powodu optymalizować scoringu przed pomiarem zapytań do bazy i startu klienta.

## UI i atlas

- Build Astro wygenerował 13 stron: główną, indeks atlasu i 11 kart. Raportowany czas builda w lokalnym przebiegu: około 0,5 s (po instalacji zależności).
- Główny bundle strony warunków: 974,7 KiB / 255,2 KiB gzip. Jest w nim statycznie importowany MapLibre; to kandydat do osobnego pomiaru i ładowania na żądanie. Nie zmieniono progu ostrzegania bundlera, żeby ukryć ten wynik.
- Łącznie 32 referencjonowane SVG atlasu: 3519,3 KiB. Nowe portrety są wektorami bez zewnętrznych zasobów.
- Rozmiary assetów i gzip raportuje `npm run perf:assets`; sumy wszystkich plików nie są transferem pojedynczej strony.

## Weryfikacja i dalsze pomiary

Przeszły testy Go z detektorem wyścigów, walidacja atlasu i build UI. Końcowy fuzzing wykonał 5 667 452 próby w 10 s bez awarii. Każdy taki przebieg jest ograniczonym eksperymentem, nie dowodem bezpieczeństwa.

Zweryfikowano CLI doctor i dostępność Docker daemon. W tej zmianie nie uruchamiano całego okna Tauri ani nie mierzono startu desktopu, FPS, baterii, rzeczywistych geometrii BDL, SQL/gRPC lub transferu map. Kolejny pomiar powinien oddzielić zimny start, ciepły start, opóźnienie API P50/P95 oraz działanie bez internetu.

## Podpisane paczki — pomiar 2026-10-02

Apple M4, macOS/arm64, Go 1.27.1. Mediany trzech przebiegów, bez równoległego fuzzingu lub kompilacji. Implementacja i zakres kontraktu: [PACK-FORMAT.md](PACK-FORMAT.md).

```sh
cd oentike-api
go test ./internal/pack -run '^$' -bench . -benchmem -count=3
```

| Operacja | Czas | Pamięć Go | Alokacje |
|---|---:|---:|---:|
| Ed25519 + ścisły JSON + walidacja manifestu jednego pliku | 28 833 ns | 4062 B | 75 |
| Inwentarz + rozmiar + SHA-256 jednego pliku 16 MiB przez mmap | 5 421 474 ns | 3712 B | 31 |

Drugi pomiar obejmuje otwarcie/zamknięcie katalogu i pliku, mapowanie oraz odmapowanie. Fixture to 16 MiB zer zapisanych przed pomiarem, z ciepłym cache systemu plików; przepustowość około 3095 MB/s nie jest prędkością dysku ani transferu sieciowego. Ed25519 nie jest częścią drugiego benchmarku. Strony mmap i całkowity RSS nie wchodzą do licznika pamięci Go. Nie mierzono zimnego cache, page faults, RSS, telefonu ani zużycia energii; brak podstaw do deklarowania zysku względem odczytu buforowanego.

To jednorazowa kontrola importu, z alokacjami parsera i inwentarza. Cel 0 allocs/op dla rozgrzanych zapytań przestrzennych pozostaje oddzielnym wymaganiem. Nie dodano zależności modułu Go ani `unsafe` w nowym kodzie.

Przeszło `go test -race ./...` całego API, w tym CLI i negatywne testy integralności. `FuzzParse` z budżetem `-fuzztime=10s` wykonał 1 236 071 prób; cały raportowany przebieg trwał 11,310 s, bez awarii. Fuzzing nie stanowi dowodu bezpieczeństwa. Nie uruchamiano weryfikatora Rust/Tauri, ponieważ nie jest częścią tego zakresu.

## Importer Rust/Tauri — weryfikacja 2026-10-03

macOS/arm64, Rust/Cargo 1.98.0. `cargo test --locked --all-targets` przeszedł: 11 testów modułu paczek, w tym wspólny wektor z Go, mutacje parsera, izolacja prywatnej kopii, błędy przed atomowym commit, antyrollback i utrzymanie starego mmap przy aktualizacji do innych bajtów zasobu. Ponowne otwarcie magazynu testowano bez sieci i API, bez uruchamiania GUI Tauri.

`cargo clippy --locked --all-targets -- -D warnings -A clippy::result_large_err` kontroluje kod z wyjątkiem ostrzeżenia o rozmiarze istniejącego `tonic::Status` w wygenerowanych metodach gRPC. Zmiana parsera Go odrzucająca `-0` przeszła testy paczki i CLI z `-race`. Kolejny `FuzzParse -fuzztime=10s` wykonał 1 886 249 prób, bez awarii (cały przebieg 11,481 s). Deterministyczny test mutacji w Rust nie jest sesją libFuzzer.

Import używa jednego bufora kopiowania 64 KiB, a SHA-256 liczy przez mmap prywatnych plików po zamknięciu ich zapisu. Alokacje metadanych, wpisów katalogów i ścieżek pozostają; nie deklarujemy 0 allocs dla całego importu. Kopiowanie wejścia jest zamierzone: nie mapujemy plików, które nadawca może skrócić w trakcie odczytu. Manifest jest ograniczony do 1 MiB, zasoby do 8 GiB na plik i 32 GiB łącznie. Odczyt aktywnego wydania ponownie hashuje wszystkie zasoby, więc jego koszt rośnie z rozmiarem paczki.

Nie mierzono jeszcze czasu/RSS natywnego importu ani kosztu fsync, zimnego startu, odczytu aktywnej dużej paczki i baterii telefonu. Nie przenosimy wyników benchmarków Go na implementację Rust. Następny pomiar powinien rozdzielić kopiowanie, hashowanie i synchronizację dysku na rzeczywistej paczce. Granice implementacji i trwałości: [PACK-IMPORT.md](PACK-IMPORT.md).
