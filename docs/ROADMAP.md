# Oentike: rozwój od obecnego fundamentu

Szczegółowy kierunek architektury, kolejność wdrożeń i kryteria odbioru: [plan inżynieryjny](MASTER-PLAN.md). Dokument rozdziela wymagania od eksperymentów i funkcji już wdrożonych.

## Etap 1 — powtarzalność i jakość (ta zmiana)

- Czytnik OENT: walidacja nagłówka, indeksu, zakresów, współrzędnych i cyklu życia; brak `unsafe`. Plik pozostaje mapowany, indeks jest dekodowany raz.
- Syntetyczne testy i benchmarki bez PostGIS i ręcznie tworzonego `forests.bin`; race detector i fuzzing parsera.
- Atlas: 11 kart, manifest jako źródło listy stron, kontrola JSON Schema, odnośników i zgodności źródłowych/publicznych SVG. Cztery nowe karty mają cytacje i autorskie schematyczne portrety.
- Uruchamianie: `./oentike`, diagnostyka, instalacja z lockfile poza codziennym startem; Node i Task w mise.

## Etap 2 — kompletna paczka wyprawy (najbliższy cel funkcjonalny)

Wdrożone fragmenty: [manifest v1 i CLI Go](PACK-FORMAT.md) oraz [weryfikator i atomowy importer Rust/Tauri](PACK-IMPORT.md), z kontrolą Ed25519/SHA-256, wspólnym wektorem, antyrollback i testami awarii. Do wykonania pozostają walidacja zawartości formatów, ekran importu, pełna geometria i użycie paczki przez renderer offline. Import w aplikacji wymaga wbudowania publicznego klucza produkcyjnego przy kompilacji.

Istniejące Go/PostGIS przygotowują dane w domu. Klient czyta je lokalnie bez uruchomionej bazy i serwera API. Zachowujemy istniejący interfejs Astro/Tauri; nie przesądzamy jeszcze platformy terenowej przed sprawdzeniem pracy sensorów w tle.

Zakres pierwszej pionowej implementacji:

1. Jeden obszar, jego geometria, snapshot warunków z czasem pobrania, karty atlasu i pełne zasoby mapy.
2. Manifest z wersją formatu, źródłami/licencjami, rozmiarami i sumami kontrolnymi. Hash wykrywa uszkodzenie; autentyczność wymaga podpisu z zaufanym kluczem.
3. Import do katalogu tymczasowego, limity rozpakowania, walidacja i atomowe przełączenie. Nie nadpisujemy otwartego pliku mmap.
4. Pełna topologia (wielopoligony i otwory) oraz porównanie zapytań z referencją PostGIS. Dzisiejszy eksporter `tools/exporter` bierze pierwszy poligon i pierścień zewnętrzny: nie wystarcza do takiej paczki.
5. Test akceptacyjny: restart aplikacji przy wyłączonym internecie i API, otwarcie obszaru, mapy, atlasu i historycznych warunków. Brak sieci nie oznacza świeżości prognozy.

Dane mapowe muszą pochodzić ze źródła dopuszczającego przygotowywanie paczek offline. Obecne publiczne endpointy rastrowe nie są automatycznie zgodą na masowe pobieranie.

## Etap 3 — użyteczny tryb terenowy

- Lokalny parking, dziennik wyprawy, zapis śladu odporny na restart, eksport na żądanie.
- Model wysokościowy, profil trasy i widok 3D. Najpierw teren i własne oznaczenia przeszkód.
- Wariant powrotu własnym śladem oraz routing z ujawnionymi ograniczeniami danych.
- Pomiar baterii, odtwarzanie zapisów sensorów i kontrolowane przerwy GNSS.

## Etap 4 — jakość szacowania powrotu

Niepewność lokalizacji i kosztu przejścia, przedziały czasu powrotu, porównanie z routingiem deterministycznym. Kryteria: kalibracja przedziałów, błędy czasu przejścia, pamięć i energia. Dopiero z odpowiednim zbiorem danych: lokalny model przechodniości.

Mesh i kryptograficzne dzielenie się punktami to kolejne moduły. Każdy wymaga własnego modelu zagrożeń i eksperymentu; nie są warunkiem dostarczenia pierwszej paczki.

## Równoległy rozwój atlasu

- Uzupełnić źródła siedmiu starszych kart i sprawdzić polskie nazwy, status prawny oraz regionalną sezonowość.
- Przegląd człowieka: morfologia, porównania oraz ilustracje. `reviewed_at` pozostaje null aż do tego przeglądu.
- Nowe gatunki dodawać partiami domykającymi istniejące porównania. Pilnować wspólnej palety, ale kształt i hymenofor muszą wynikać z gatunku.
- Dodać wielojęzyczne treści kart: obecny przełącznik zmienia nazwy/etykiety, opis pozostaje po polsku.
- Przed dystrybucją zamrozić wersję paczki i ustalić licencje kodu, treści, własnych SVG i danych zewnętrznych. Obecny `pilot-0.0.1` pozostaje roboczą, niepodpisaną paczką.
