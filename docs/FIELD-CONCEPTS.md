# Oentike — cztery kierunki terenowe

Propozycja architektury z 2026-09-24. To projekt rozwoju, nie opis wdrożonych funkcji. Punktem odniesienia są README, ROADMAP, obecny klient Astro/Tauri/MapLibre, Go/PostGIS i czytnik OENT.

Cel: osobiste narzędzie terenowe i mocne portfolio inżynierskie. Poziom pracy magisterskiej jest odniesieniem dla jakości, nie wymaganiem akademickim. Zakres wynika z użyteczności w lesie; eksperymenty i benchmarki mają uzasadniać decyzje oraz wykrywać błędy.

## Wspólny fundament

Go/PostGIS przygotowują paczkę w domu. Urządzenie terenowe czyta lokalny plik mapy i lokalny dziennik bez API, konta i Dockera. PostGIS pozostaje narzędziem przygotowania danych; SQLite przechowuje dziennik. MapLibre pozostaje rendererem. Rust obsługuje pliki i obliczenia klienta. Nie portujemy równolegle tego samego silnika do Go, Rust i WASM.

Paczka zawiera manifest, mapę PMTiles, lokalne style/fonty/sprite'y, opcjonalny DEM, geometrię obszaru, snapshot warunków z czasem pobrania oraz istniejący atlas. Import: ograniczenia rozmiarów i liczby plików, odrzucenie ścieżek wychodzących poza katalog, kontrola hashy, katalog tymczasowy i atomowa aktywacja. Podpis ma znaczenie wyłącznie z zaufanym kluczem. Żadnego nadpisywania pliku używanego przez mmap.

PMTiles jest kontenerem kafelków, nie grafem routingu. Dane topologiczne wymagają osobnego formatu. Obecny OENT reprezentuje pojedyncze pierścienie; nie wolno traktować go jako pełnej topologii wielopoligonów z otworami.

Desktop służy także do odtwarzania wypraw. Przed zobowiązaniem się do terenowego klienta Tauri trzeba wykonać próbę na docelowym telefonie: GNSS przy zablokowanym ekranie, restart procesu, zużycie baterii i trwałość zapisu. Sam działający widok w WebView nie dowodzi działania w tle.

## 1. Return Envelope — powrót z jawną niepewnością

**Problem:** odległość do samochodu nie opisuje kosztu powrotu przez rów, zbocze i gęstwinę.

**Przepływ:** zapis parkingu → trwały ślad → powrót po własnym śladzie → alternatywy na grafie terenu → przedział czasu powrotu i izochrony w 3D.

**Mechanika:** graf skierowany; koszt przejścia zależy od kierunku i nachylenia. DEM i przeszkody przygotowane przez GDAL/PostGIS. Dijkstra daje punkt odniesienia. A* dopiero z udowodnioną dolną granicą kosztu; heurystyka zero pozostaje poprawna. Nieznane przejście nie staje się drogą. Scenariusze prędkości i opóźnień dają empiryczny rozkład czasu, kalibrowany na oddzielnych wyprawach. Brak danych oznacza brak wiarygodnego przedziału. MapLibre pokazuje teren, ślad, bariery i zasięg powrotu.

**WOW:** porównanie kosztu tras z najkrótszą drogą i własnym śladem; kalibracja przedziałów, p95 czasu obliczeń, pamięć, energia na godzinę. Symulator utraty GNSS i awarii procesu. Wyjaśnienie, dlaczego system wybrał obejście.

**Privacy/FOSS:** ślad lokalny, bez domyślnej analityki; szyfrowanie prywatnego dziennika, klucz w magazynie systemowym, eksport na żądanie. Model oraz procedura benchmarków otwarte. Rzeczywiste miejscówki nie trafiają do repozytorium testów.

**Pierwszy wycinek:** parking i odtwarzalny ślad z zapisem odpornym na restart. Własny ślad jest historią przejścia, nie gwarancją aktualnej drożności.

## 2. Forest Observatory — prywatne laboratorium mikroterenu

**Problem:** wynik pogodowy dla nadleśnictwa nie wyjaśnia różnic między sąsiednimi fragmentami lasu.

**Przepływ:** obserwacja terenowa → lokalne cechy topograficzne → porównanie odwiedzin → wyjaśniona mapa hipotez siedliskowych i miejsc niedostatecznie zbadanych.

**Mechanika:** GDAL liczy nachylenie i ekspozycję; dodatkowy etap hydrologiczny wyznacza akumulację odpływu i wskaźniki wilgotności topograficznej. Przetwarzanie w metrycznym CRS, jawna rozdzielczość i NoData. Raster cech oraz DEM jadą w paczce. Lokalny mały model regresyjny, najpierw jako baseline, aktualizuje się na opisanych obserwacjach. Brak znaleziska zapisujemy osobno od braku odwiedzin; rejestrujemy czas poszukiwań. Model przewiduje określoną wielkość, np. subiektywną wilgotność podłoża, a nie jadalność gatunku.

**WOW:** walidacja blokami przestrzennymi i sezonami zamiast losowego podziału pobliskich punktów, ablation study cech DEM, analiza błędu próbkowania i jakości kalibracji. Interaktywny profil terenu z wyjaśnieniem cech.

**Privacy/FOSS:** trening i inferencja lokalnie, dane prywatne oddzielone od publicznych warstw. Udostępnianie modelu nie jest automatycznie anonimowe. Publiczny benchmark wykorzystuje dane syntetyczne lub świadomie udostępnione. Licencje kodu, danych i wag rozpatrujemy osobno.

**Pierwszy wycinek:** jeden raster nachylenia, obserwacja wilgotności i wyjaśnienie bez AI. Model dopiero po zgromadzeniu danych; TWI nie jest pomiarem bieżącej wilgotności.

## 3. Quiet Rendezvous — spotkanie bez ujawniania miejscówek

**Problem:** grupa rozdziela się bez internetu; potrzebuje krótkich wiadomości i punktu spotkania, a nie wspólnej historii wszystkich znalezisk.

**Przepływ:** parowanie QR → wybór punktu spotkania → zaszyfrowana wiadomość → potwierdzenie odbioru po odzyskaniu kontaktu.

**Mechanika:** na początek transfer lokalny, następnie adapter Meshtastic przez zewnętrzne radio LoRa. Telefon sam nie zyskuje łączności LoRa. Osobny klucz wyprawy, identyfikator nadawcy, licznik wiadomości, deduplikacja, limit rozmiaru i priorytety. Uwierzytelnione szyfrowanie z biblioteki, unikalne nonce; odbiorca odróżnia wiadomość opóźnioną od aktualnej. Status dostarczenia wynika z potwierdzenia, nigdy z samego wysłania.

Automerge może obsługiwać edytowane notatki przy kontakcie o większej przepustowości. Surowy ślad pozostaje dopisywanym dziennikiem; całej historii CRDT nie przesyłamy przez LoRa. Autoryzacja i szyfrowanie są osobnym wymaganiem od scalania zmian.

**WOW:** testy partycji sieci, duplikacji, utraty i zmiany kolejności wiadomości; budżet bajtów, czas potwierdzenia, energia i model zagrożeń. Realny pomiar w lesie zamiast deklarowanego zasięgu.

**Privacy/FOSS:** udostępniamy uzgodniony punkt spotkania, nie prywatny punkt znaleziska. Szyfrowanie nie ukrywa wszystkich metadanych radiowych. Odwołanie dostępu nie usuwa danych już skopiowanych przez odbiorcę. ZKP może później dowodzić przynależności zadeklarowanego punktu do obszaru, ale bez zaufanego źródła lokalizacji nie dowodzi fizycznej obecności człowieka.

**Pierwszy wycinek:** dwie instancje, jedna wiadomość, przerwanie łączności, ponowny kontakt i dokładnie jeden logiczny zapis u odbiorcy.

## 4. Canopy Fix — uczciwa lokalizacja pod koronami

**Problem:** skoki GNSS tworzą fikcyjne pętle; nieruchomy użytkownik pozornie zmienia kierunek i dystans.

**Przepływ:** GNSS + IMU → detekcja bezruchu i odrzucanie odstających pomiarów → ślad z niepewnością → kontrolowane pogorszenie przy zaniku sygnału.

**Mechanika:** natywna rejestracja sensorów, czas monotoniczny dla odstępów i osobny czas UTC. Lokalny układ ENU, filtr stanu z prędkością i modelem błędu, detekcja bezruchu. PDR dopiero po walidacji sposobu noszenia telefonu. Dopasowanie do ścieżki jest miękką hipotezą; grzybiarz często idzie poza ścieżką. Przy braku GNSS niepewność rośnie, a system nie ukrywa tego wygładzeniem. Barometr wymaga uwzględnienia dryfu.

**WOW:** deterministyczny replay identycznych próbek, porównanie z surowym GNSS i prostym filtrem, błąd pozycji, dryf bez sygnału, odsetek fałszywych odrzuceń, kalibracja obszaru niepewności i energia. Potrzebna niezależna referencja pomiarowa; sam surowy GNSS nie jest ground truth.

**Privacy/FOSS:** sensory i modele pozostają na urządzeniu. Otwarte formaty odtwarzania i generator usterek umożliwiają ocenę bez publikowania śladów użytkownika. Dane debugowania nie zapisują domyślnie współrzędnych w logach aplikacji.

**Pierwszy wycinek:** rejestrator i porównanie surowego śladu z odfiltrowanymi skokami, z jawnym oznaczeniem luk pomiarowych.

## Kolejność i kryteria dostarczenia

1. Paczka jednego obszaru: po restarcie, bez internetu i API, działa mapa, atlas i snapshot z datą. Test przechwytujący ruch potwierdza brak zależności sieciowych.
2. Parking i dziennik: zabicie procesu nie usuwa zatwierdzonych próbek; jawny limit potencjalnej utraty ostatniej partii.
3. Return Envelope: najpierw własny ślad, potem DEM, graf i warianty kosztów.
4. Canopy Fix: pomiary na docelowym telefonie i replay przed integracją modelu niepewności z routingiem.
5. Observatory i Rendezvous jako niezależne rozszerzenia po działającym rdzeniu.

Oś produktu: nawigacja powrotna offline w środowisku leśnym z modelowaniem niepewności lokalizacji i kosztu przejścia. Model niepewności wdrażamy wtedy, gdy pomiary wykażą poprawę szacowania czasu powrotu względem prostego wariantu deterministycznego. Nie zakładamy poprawy z góry ani nie rozbudowujemy zakresu dla nowości naukowej.

Pierwszy eksperyment 60 minut: lokalny fixture parkingu i śladu → obliczenie długości powrotu po śladzie → zapis → restart → identyczny wynik. Kolejne sesje 45–90 minut wydzielamy według pojedynczego kryterium: walidacja manifestu, uszkodzony plik, import atomowy, lokalny renderer, test bez sieci. Integracja paczki nie jest obietnicą ukończenia całego etapu w godzinę.

## Grafika i uruchamianie

Referencją ilustracji są istniejące dopracowane SVG: paleta, faktura, nieregularny kontur, gęstość detali, skala i kompozycja. Wspólne kolory oraz poprawna walidacja SVG nie dowodzą zgodności stylu. Przed rozszerzeniem atlasu potrzebny render porównawczy starszej ilustracji i nowej w tej samej wielkości karty; proste schematy nie zastępują jakości referencji. Morfologia podlega osobnej kontroli. Ta propozycja nie zmienia ilustracji.

Istniejący launcher `./oentike` pozostaje wejściem developerskim. Docelowy użytkownik terenowy uruchamia zainstalowaną aplikację i importuje paczkę. Nie wymaga to Go, Node, PostGIS ani Dockera na telefonie.

Poza pierwszą wersją: SLM do nawigacji, publiczny IPFS, własne protokoły kryptograficzne, ZKP i pełny mesh. Najpierw mierzalny przepływ offline.

## Źródła techniczne

- PMTiles: https://docs.protomaps.com/pmtiles/
- MapLibre DEM: https://maplibre.org/maplibre-style-spec/sources/
- GDAL: https://gdal.org/en/stable/programs/gdaldem.html
- Automerge, synchronizacja: https://automerge.org/docs/reference/concepts/
- Meshtastic: https://meshtastic.org/docs/introduction/
- Libsodium AEAD: https://doc.libsodium.org/secret-key_cryptography/aead/chacha20-poly1305
