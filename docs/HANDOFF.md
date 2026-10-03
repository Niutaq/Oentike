# Oentike — stan do samodzielnej kontynuacji

Zapis: **2026-10-03**, na prośbę właściciela projektu przed zakończeniem pracy asystenta. Ten plik pozwala kontynuować bez historii rozmowy. Nie uruchamia żadnych dalszych prac.

## Repozytorium i stan Git

- Katalog roboczy: `/Users/niutaq/Documents/Project Data/Oentike`.
- Gałąź: `dev_0005`, śledząca `origin/dev_0005`.
- HEAD podczas przekazania: `c16e754` — `Update spatial analysis, ingest loop, and UI for Atlas`.
- Zmiany z tej rozmowy są **niezacommitowane**. Nie utworzono PR, nie wykonano push ani wdrożenia.
- Worktree już przed rozpoczęciem prac zawierało dużo zmian. Są w nim m.in. poprawki OENT, atlasu i SVG, UI, lokalnej wyprawy, launchera `oentike`, skryptów, Taskfile i mise. Zachowano je; nie należy przypisywać całego diffu importerowi.
- Całe `docs/`, nowe pakiety Go oraz `src-tauri/src/packs/` są obecnie nieśledzone. Sam `git diff` ich nie pokaże — sprawdzaj również `git status --short` i treść nowych plików. Przed commitem wybierz świadomie zakres plików.

## Co rzeczywiście mamy

Fundament istniejący przed tą rozmową: backend Go/PostGIS z BDL i pogodą Open-Meteo, gRPC, Astro/Tauri, atlas oraz czytnik OENT przez mmap bez `unsafe` w kodzie Go. OENT nadal przechowuje pojedyncze pierścienie, bez pełnej obsługi wielopoligonów i otworów. Istniejące pliki trybu wyprawy nie zostały w tej rozmowie ponownie ocenione ani przetestowane jako cały produkt.

W tej rozmowie dostarczono dwa zakresy:

| Zakres | Stan |
|---|---|
| Plan inżynieryjny | Zapisany w `MASTER-PLAN.md`, połączony z roadmapą |
| Manifest paczki v1 | Ścisły kontrakt JSON, limity, Ed25519 i SHA-256 zasobów |
| Generator/weryfikator Go | Gotowe CLI `oentike-pack sign` i `verify` |
| Wspólny wektor Go/Rust | Jedna fixture, sprawdzana po obu stronach |
| Weryfikator Rust | Podpis, ścisły JSON, tożsamość obszaru, metadane i hashe |
| Importer natywny Tauri | Kopiowanie do prywatnego magazynu i atomowa aktywacja przez wskaźnik |
| Ochrona aktualizacji | Odrzucanie starszych/równych wydań, blokada równoległego importu, zachowanie starych plików mmap |
| Ekran importu | Jeszcze nie ma; dostępne są zarejestrowane komendy natywne |
| Kompletna mapa offline | Jeszcze nie ma; zasoby paczki nie są podłączone do renderera |

To podstawa dystrybucji zaufanych zasobów, a nie ukończony tryb terenowy.

## Pliki dodane lub zmienione w ramach tej rozmowy

| Ścieżka od korzenia repozytorium | Rola |
|---|---|
| `oentike-api/internal/pack/manifest.go` | Schemat, walidacja JSON, podpis i weryfikacja Ed25519 |
| `oentike-api/internal/pack/files.go` | Budowanie inwentarza i kontrola kompletności zasobów |
| `oentike-api/internal/pack/hash_unix.go` | Hashowanie przez mmap na Linux/macOS |
| `oentike-api/internal/pack/hash_other.go` | Jawny błąd na pozostałych platformach |
| `oentike-api/internal/pack/*_test.go` | Testy, fuzzing parsera i benchmarki |
| `oentike-api/internal/pack/testdata/vector.json` | Stały wektor; zawiera publicznie znany seed **wyłącznie testowy** |
| `oentike-api/cmd/oentike-pack/` | CLI oraz jego testy |
| `oentike-web/src-tauri/src/packs/manifest.rs` | Weryfikacja i walidacja kontraktu w Rust |
| `oentike-web/src-tauri/src/packs/store.rs` | Magazyn, prywatna kopia, mmap, blokada, commit i ponowny odczyt |
| `oentike-web/src-tauri/src/packs/mod.rs` | Konfiguracja zaufania i komendy Tauri |
| `oentike-web/src-tauri/src/packs/tests.rs` | Testy zgodności, błędów, mutacji parsera i cyklu życia importu |
| `oentike-web/src-tauri/src/lib.rs` | Rejestracja dwóch nowych komend |
| `oentike-web/src-tauri/build.rs` | Śledzenie zmiennej klucza przy kompilacji |
| `oentike-web/src-tauri/Cargo.toml`, `Cargo.lock` | Zależności importera; lockfile zachował istniejące wersje i dodał nowe wpisy |
| `docs/MASTER-PLAN.md`, `ROADMAP.md`, `PERFORMANCE.md` | Kierunek, stan wdrożenia i wyniki weryfikacji |
| `docs/PACK-FORMAT.md`, `PACK-IMPORT.md`, `HANDOFF.md` | Kontrakt, obsługa importera i to przekazanie |

Nowe bezpośrednie zależności Rust: `ed25519-dalek =2.2.0`, `sha2 =0.10.9`, `memmap2 =0.9.9`, `tempfile =3.27.0`, Unix: `rustix =1.1.4` z funkcją `fs`. Go nie otrzymało nowych zależności modułu.

## Najważniejsze ustalenia kontraktu

- Podpis obejmuje `ASCII("OENTIKE-PACK-MANIFEST-V1") || 0x00 || dokładne bajty manifest.json`. To zwykły Ed25519; podpis jest osobnym plikiem 64 surowych bajtów. Nie wolno normalizować JSON przed sprawdzeniem podpisu.
- Manifest obejmuje SHA-256 i rozmiar każdego zasobu. Oddzielny podpis każdego pliku nie jest wymagany.
- Limity: manifest 1 MiB, do 4096 plików, do 8 GiB na plik i 32 GiB łącznie. Teksty metadanych i ścieżki mają ograniczony alfabet ASCII; treści samych zasobów mogą zawierać Unicode.
- Duplikaty pól, brakujące/nieznane pola, aliasy wielkości liter, traversal, kolizje nazw i nieprawidłowe liczby są odrzucane. Podczas integracji Rust doprecyzowano odrzucenie `-0` również w Go, z testem regresyjnym.
- Szablon CLI Go ma pełny schemat. `size` można ustawić na 0, `sha256` na 64 zera; generator je przelicza. Katalog zasobów i katalog metadanych są oddzielne.
- Fixture zawiera syntetyczny zasób `hello` + LF. Nie jest rzeczywistą paczką BDL, pogodą ani kompletną mapą.

Pełny kontrakt i polecenia CLI: [PACK-FORMAT.md](PACK-FORMAT.md).

## Klucz i użycie Tauri

**Nie wygenerowano ani nie skonfigurowano klucza produkcyjnego.** Import w zwykłym buildzie bez klucza pozostaje celowo zablokowany.

Podczas kompilacji Tauri ustaw `OENTIKE_PACK_PUBLIC_KEY_HEX` na publiczny klucz Ed25519 zapisany jako 64 małe cyfry hex. To wartość wbudowana w binarkę przez `option_env!`, nie konfiguracja odczytywana podczas działania. `build.rs` wymusza reakcję na zmianę zmiennej. Seed prywatny pozostaje poza repozytorium i aplikacją. Klucz z fixture jest jawnie odrzucany przez konfigurację aplikacji; bezpośrednie testy wewnętrznego magazynu mogą go używać.

Zarejestrowane komendy:

```ts
await invoke('import_offline_pack', {
  areaId: 'nadl-05-31',
  payloadDir: '/absolute/path/to/payload',
  manifestPath: '/absolute/path/to/metadata/manifest.json',
  signaturePath: '/absolute/path/to/metadata/manifest.sig',
});
await invoke('get_offline_pack', { areaId: 'nadl-05-31' });
```

Klucz i katalog docelowy nie są argumentami IPC. Magazyn: `app_data_dir()/offline-packs-v1`. Ciężkie operacje działają przez `spawn_blocking`. Szczegóły wyniku i struktury dysku: [PACK-IMPORT.md](PACK-IMPORT.md).

## Co zostało sprawdzone

Środowisko wykonania: macOS/arm64, Apple M4, Go 1.27.1, Rust/Cargo 1.98.0. Poniższe wyniki pochodzą z poprzednich kroków tej rozmowy; podczas zapisywania tego przekazania testów ponownie nie uruchamiano.

- Po wdrożeniu Go przeszło `go test -race ./...` całego API.
- Po ujednoliceniu `-0` przeszło `go test -race ./internal/pack ./cmd/oentike-pack`.
- Ostatni Go `FuzzParse -fuzztime=10s`: **1 886 249 prób bez awarii**, 11,481 s łącznie z zakończeniem przebiegu.
- Rust `cargo test --locked --all-targets`: **11 testów modułu paczek przeszło**, testowy target binarki również przeszedł (0 testów).
- `cargo clippy --locked --all-targets -- -D warnings -A clippy::result_large_err` przeszło. Bez tego wyjątku Clippy zgłasza duży `tonic::Status` w istniejącym wygenerowanym gRPC. Nie zmieniano generatora, żeby ukryć ostrzeżenie.
- Kontrole formatowania nowych plików i `git diff --check` przeszły.
- Test Rust sprawdza identyczny podpis i serializację wspólnego wektora Go, błędne podpisane JSON-y i deterministyczne mutacje. Nie uruchamiano Rust libFuzzer.
- Testy importera sprawdzają m.in. błędne hashe, symlinki/hardlinki/FIFO, antyrollback, inną tożsamość obszaru, blokadę równoległego importu, błędy przed commit, odtworzenie stanu z dysku oraz stare mmap z poprzednimi bajtami po instalacji nowego wydania.

Polecenia do powtórzenia z odpowiednich katalogów:

```sh
# Z oentike-api/
go test -race ./internal/pack ./cmd/oentike-pack
go test ./internal/pack -run '^$' -fuzz FuzzParse -fuzztime=10s
go test ./internal/pack -run '^$' -bench . -benchmem -count=3
```

```sh
# Z oentike-web/src-tauri/
cargo test --locked --all-targets
cargo clippy --locked --all-targets -- -D warnings -A clippy::result_large_err
```

Benchmarki Go z 2026-10-02: mediana małego manifestu 28,833 µs / 4062 B / 75 alokacji; kontrola 16 MiB zasobu przez mmap 5,421 ms / 3712 B / 31 alokacji, ciepły cache. To nie są wyniki telefonu, dysku ani importera Rust. Szczegóły i ograniczenia: [PERFORMANCE.md](PERFORMANCE.md).

## Ograniczenia i sprawy otwarte

1. Atomowa aktywacja potwierdza podpisany inwentarz i bajty, nie poprawność geometrii/PMTiles, kompletność mapy ani świeżość pogody. Walidacja zawartości formatów nadal wymaga implementacji.
2. Importer przyjmuje lokalne katalogi. Nie ma archiwum transportowego, pobierania, wznowień ani interfejsu wyboru plików w UI.
3. Kod importera jest Unixowy. Przetestowano macOS; Linux i urządzenia mobilne wymagają sprawdzenia. Windows zwraca błąd nieobsługiwanej platformy.
4. Prywatne pliki są kopiowane z buforem 64 KiB, dopiero potem mapowane. Rust ma jeden blok `unsafe` w kodzie produkcyjnym mapowania z opisanym warunkiem niezmienności. Obcy proces tego samego użytkownika/administrator może naruszyć magazyn; to nie jest ochrona przed takim przeciwnikiem.
5. Stare wydania, kompletne osierocone wydania oraz staging po nagłym przerwaniu procesu nie są automatycznie usuwane. Sprzątanie i limity retencji pozostają osobnym zadaniem.
6. `active.json` jest punktem commit. Błąd przed commit zachowuje stare wydanie. Po commit wynik może mieć `durabilityConfirmed: false`, jeżeli końcowy fsync zawiedzie — nowa paczka jest już wtedy aktywna. Nie testowano fizycznego zaniku zasilania.
7. Odczyt `get_offline_pack` ponownie hashuje wszystkie zasoby. Koszt rośnie z rozmiarem paczki; nie traktować tej komendy jako taniego odpytywania co klatkę.
8. Nie uruchamiano pełnego GUI Tauri z importem, nie wykonywano testu mapy bez internetu, pomiaru baterii, RSS importera ani benchmarku Rust. Odtworzenie magazynu z dysku w teście nie zastępuje restartu całej aplikacji.
9. Rotacja/unieważnianie kluczy i lista wielu kluczy zaufania nie są zaimplementowane. Antyrollback opiera się na zachowanym lokalnym stanie aktywnej paczki, nie na zewnętrznym liczniku odpornym na ręczne usunięcie danych.
10. `Cargo.toml` nadal deklaruje `rust-version = "1.71"`; ta deklaracja nie została zweryfikowana i wymaga uzgodnienia z zależnościami (np. dodany `ed25519-dalek 2.2.0` deklaruje Rust 1.81). `mise.toml` i faktycznie użyte narzędzia również mają różne wersje. Nie traktować udanych testów na Rust 1.98 jako potwierdzenia starszego MSRV.

## Kolejność dalszych prac

Do wyboru przy samodzielnej kontynuacji:

1. Przejrzeć i utrwalić obecny zakres w Git, uwzględniając nowe nieśledzone pliki i wcześniejsze niezależne zmiany. Uzgodnić wersje toolchaina/MSRV.
2. Ustalić własny klucz podpisujący poza repozytorium i wbudować publiczny klucz w klienta. Nie używać seeda z fixture do dystrybucji.
3. Dodać ekran importu i czytelne statusy, korzystając z istniejących komend. Przed użytkowym renderowaniem dołączyć walidatory formatów i przygotować rzeczywistą paczkę jednego obszaru.
4. Domknąć pełną topologię geometrii, lokalne style/fonty/sprite'y i mapę. Test odbioru: restart aplikacji przy odciętej sieci i API, działająca mapa, atlas oraz jawnie datowane warunki.
5. Zaplanować retencję wydań i pomiary rzeczywistego importu, startu oraz pamięci. Sprzątanie musi respektować otwarte czytniki.

dRPC, FlatBuffers/GeoParquet, JetStream, ONNX, scraper, DigitalOcean/Terraform/Infracost i Sentry pozostają **planami lub eksperymentami**, nie wdrożeniami z tej rozmowy. Istniejącego gRPC nie migrowano. Kierunek i kryteria odbioru: [MASTER-PLAN.md](MASTER-PLAN.md), [ROADMAP.md](ROADMAP.md). Prywatność pozostaje wymaganiem: parking, ślad i dziennik lokalnie, bez telemetrii lokalizacyjnej.

Instrukcje dla pracy w `oentike-web/` znajdują się w `oentike-web/AGENTS.md`; należy je przeczytać przed zmianami UI. Stan zapisany — dalsze decyzje i prace przejmuje właściciel projektu.
