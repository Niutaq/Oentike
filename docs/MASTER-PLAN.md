# Oentike — plan inżynieryjny

Kierunek przyjęty 2026-10-02 dla `dev_0005`. Dokument określa prace docelowe, nie potwierdza ich wdrożenia. Kolejność etapów funkcjonalnych pozostaje w [ROADMAP.md](ROADMAP.md), wyniki pomiarów w [PERFORMANCE.md](PERFORMANCE.md).

## Fundament i granice

Go/PostGIS przygotowują paczki; Astro/Tauri korzysta z lokalnych danych podczas wyprawy. Aktualne RPC to cztery metody unary gRPC w `oentike-proto/conditions.proto`; streaming paczek wymaga nowego kontraktu. OENT udostępnia wierzchołki przez mmap bez `unsafe` w czytniku Go, ale nie obsługuje otworów ani wielopoligonów. Istniejące wyniki wydajności dotyczą syntetycznych fixture na desktopie.

Parking, ślad i dziennik pozostają lokalne. Eksport jest świadomą operacją użytkownika. Predykcja warunków nie identyfikuje gatunku ani nie potwierdza bezpieczeństwa spożycia grzyba.

## Kolejność dostarczenia

| Krok | Rezultat | Kryterium odbioru |
|---|---|---|
| 1. Zaufana paczka | Specyfikacja manifestu, podpisujący generator Go, weryfikator i importer Rust/Tauri | Wspólne wektory testowe Go/Rust; odrzucenie modyfikacji manifestu, danych i podpisu; awaria importu zachowuje poprzednią paczkę |
| 2. Pełna geometria | Wersjonowany format wielopoligonów i otworów, eksport i czytnik mmap | Wyniki zgodne z ustaloną semantyką granicy w PostGIS; fuzzing; pomiary czasu otwarcia, RSS i zapytań |
| 3. Pionowy tryb offline | Jeden obszar, mapa i jej zasoby, atlas, snapshot pogody i warunków | Restart z odciętą siecią i wyłączonym API; kompletna mapa i jawny czas aktualizacji |
| 4. Synchronizacja dRPC | Kontrakt transferu oraz działający klient w docelowym środowisku Tauri | Przerwany transfer, wznowienie, anulowanie, limity pamięci i zgodność wersji; porównanie z obecną ścieżką |
| 5. Eksperyment JetStream | Idempotentny worker pogodowy z odtwarzaniem | Powtórzona wiadomość nie duplikuje danych; awaria po zapisie przed ACK nie zmienia wyniku |
| 6. Predykcja lokalna | Model ONNX z wersją cech i lokalnym fallbackiem | Walidacja czasowa i przestrzenna, porównanie z heurystyką oraz pomiar RAM, opóźnienia i energii na urządzeniu |

Infrastruktura chmurowa i eksperymenty nie blokują kroków 1–3.

## Kontrakt bezpieczeństwa paczki

- Manifest zawiera wersję schematu, identyfikator obszaru i wydania, czas utworzenia, wersje formatów oraz listę plików z rozmiarem, SHA-256 i źródłami/licencjami. Snapshot pogody ma osobny czas pobrania i zakres dat.
- Ed25519 podpisuje dokładne bajty manifestu z ustalonym prefiksem domenowym protokołu. Podpis jest oddzielnym plikiem. Hash każdego zasobu jest objęty podpisem manifestu; osobny podpis każdego zasobu nie jest wymagany przez ten projekt kontraktu.
- Klucze publiczne i ich identyfikatory pochodzą z aplikacji. Klucz dostarczony przez paczkę nie ustanawia zaufania. Prywatny klucz podpisujący pozostaje poza repozytorium i klientem. Rotacja wymaga zaufanej aktualizacji listy kluczy; offline nie zapewnia natychmiastowego unieważnienia.
- Przed parsowaniem ograniczamy długość manifestu i podpisu. Po weryfikacji podpisu wymagamy jednoznacznego JSON: odrzucamy powtórzone klucze, nieobsługiwane wersje oraz nieprawidłowe rozmiary i ścieżki.
- Importer odrzuca traversal, ścieżki absolutne, symlinki, duplikaty i kolizje nazw oraz zasoby nieobjęte manifestem. Obowiązują limity liczby plików, rozmiaru po rozpakowaniu i zajętości dysku.
- Weryfikacja hashy i struktury odbywa się przed aktywacją, w prywatnym katalogu importu. Aktywacja przełącza wydanie atomowo na tym samym systemie plików. Nigdy nie zapisujemy do pliku aktualnie używanego przez mmap; stare wydanie usuwamy po zwolnieniu czytników.
- Poprawny podpis nie zastępuje walidacji parsera ani nie gwarantuje świeżości. Zwykła aktualizacja odrzuca starszy numer wydania; archiwalne snapshoty są odrębną, jawnie oznaczoną funkcją.

Pierwszy PR implementacyjny: specyfikacja bajtowa i fixture podpisanej minimalnej paczki, generator Go oraz testy negatywne. Następny: Rust/Tauri i atomowy importer. Żaden z tych fragmentów osobno nie oznacza gotowego trybu offline.

Stan 2026-10-02: pierwszy zakres jest dostępny lokalnie jako [manifest v1 i CLI Go](PACK-FORMAT.md). Nie powstał jeszcze PR ani importer Tauri. Klient Rust musi potwierdzić zgodność ze wspólnym wektorem.

Stan 2026-10-03: [importer Rust/Tauri](PACK-IMPORT.md) potwierdza wspólny wektor, kopiuje zasoby do prywatnego magazynu, sprawdza je przez mmap i atomowo aktywuje wydanie. Blokuje obce obszary, starsze wydania i równoległy import. Klucz produkcyjny, UI importu, walidacja zawartości formatów i pełny test terenowy pozostają do dostarczenia.

## Format i wydajność

FlatBuffers jest kandydatem na format urządzenia; GeoParquet na wymianę i analitykę serwerową. Parquet stosuje kodowanie i kompresję stron, więc samo mmap nie gwarantuje odczytu geometrii bez dekodowania. Wybór formatu urządzenia wymaga prototypu i pomiarów, nie deklaracji zero-copy. Źródła: [kodowania Parquet](https://parquet.apache.org/docs/file-format/data-pages/encodings/), [kompresja Parquet](https://parquet.apache.org/docs/file-format/data-pages/compression/), [odczyt FlatBuffers w Go](https://flatbuffers.dev/languages/go/).

Duże pliki lokalne muszą mieć ścieżkę odczytu mmap. Transfer i zapis korzystają z ograniczonych buforów. Walidacja offsetów, długości, przepełnień i topologii poprzedza zapytania. Brak `unsafe` w naszym czytniku Go pozostaje wymaganiem; własności bibliotek Rust i ich granice bezpieczeństwa podlegają osobnej ocenie.

Cel 0 allocs/op dotyczy rozgrzanych pętli zapytań geometrii; alokacje inicjalizacji i wyników mierzymy oddzielnie. Raportujemy także RSS, page faults, zimny/ciepły start, transfer i P50/P95. Nie utożsamiamy mmap z zerowym zużyciem RAM ani braku alokacji Go z brakiem kopiowania w całej aplikacji.

Renderowanie wektorowe na GPU rozwijamy po pomiarze obecnego renderera i kosztu granicy WebView/native. Zmiana renderera wymaga dowodu poprawy FPS, pamięci lub energii na urządzeniu docelowym.

## Transport i przetwarzanie pogody

dRPC jest docelowym kierunkiem migracji; mniejszy narzut w Oentike pozostaje hipotezą do zmierzenia. Trzeba sprawdzić klienta Rust/Tauri, generowanie kodu, obsługę błędów, TLS i anulowanie przed usunięciem gRPC. [Projekt Storj dRPC](https://github.com/storj/drpc) jest punktem odniesienia dla prototypu.

Kontrakt synchronizacji określi maksymalny rozmiar porcji, identyfikator niezmiennego wydania, offset wznowienia i weryfikację kompletnej paczki. Serwer i klient muszą stosować backpressure i limity współbieżności.

JetStream pozostaje eksperymentem serwerowym. Wiadomości reprezentują zadania lub odnośniki do snapshotów, a nie gigabajtowe pliki. Worker zapisuje wynik trwale przed ACK, używa klucza idempotencji, ogranicza retry i respektuje limity dostawcy. Replay opiera się na utrwalonych, wersjonowanych wejściach; ponowne pobranie aktualnej pogody nie odtwarza historycznego snapshotu. Retencję i koszt przestrzeni ustalamy przed ingestem całej Polski.

## Model i dane społecznościowe

Najpierw definiujemy obserwowaną zmienną celu i zbiór etykiet. Pogoda sama nie stanowi etykiety występowania grzybów. Model XGBoost/ONNX musi pokonać wersjonowaną heurystykę na danych spoza czasu i regionu treningowego; raportujemy braki danych i niepewność. Schemat cech, preprocessing i model są wersjonowane razem i objęte podpisanym manifestem. Nieobsługiwany model lub brak cech daje jawny fallback albo brak wyniku.

Heuristic Scraper to osobny eksperyment: udokumentowane źródła i uprawnienia do użycia danych, usunięcie identyfikatorów i agregacja przestrzenna. Samo usunięcie nicka nie anonimizuje dokładnych współrzędnych i czasu. Sygnały z forów nie stają się automatycznie etykietami treningowymi; oceniamy selekcję obserwacji, błędy i manipulacje. Lokalnych śladów użytkowników nie używamy do treningu.

## Utrzymanie i FOSS

DigitalOcean to wskazany cel wdrożenia. Terraform i raport Infracost obejmują compute, dyski, backupy oraz transfer; kredyty i ich wygaśnięcie sprawdzamy przed provisioningiem. Estymacja kosztu nie jest twardym limitem rachunku. TimescaleDB dodajemy po pomiarze potrzeb historii pogodowej.

Sentry integrujemy z kontrolą danych wychodzących: bez współrzędnych, śladów, treści dziennika i payloadów paczek, z testem filtrowania zdarzeń. Dostępność pakietów studenckich wymaga sprawdzenia przy wdrożeniu. Licencje kodu, danych, atlasu i modeli dokumentujemy oddzielnie; zależności i narzędzia generatora mają przypięte wersje.

## Warunki zakończenia każdej zmiany

- Mały, kompletny zakres z testem zachowania i jawnymi ograniczeniami.
- Fuzzing nowych parserów, negatywne testy importu i testy cyklu życia mmap tam, gdzie zmiana ich dotyczy.
- Pomiary według `PERFORMANCE.md` dla zmian deklarujących korzyść wydajnościową; bez dopisywania niezmierzonych wyników.
- Test bez sieci dla funkcji terenowych oraz kontrola, że dane osobiste pozostają lokalne.
