# Podpisany manifest paczki — v1

Status: generator/weryfikator Go oraz weryfikator i importer Rust/Tauri, 2026-10-03. Implementacje: `oentike-api/internal/pack`, CLI `oentike-api/cmd/oentike-pack`, `oentike-web/src-tauri/src/packs`. To kontrakt autentyczności i integralności zasobów. Import lokalnego katalogu oraz atomową aktywację opisuje [PACK-IMPORT.md](PACK-IMPORT.md); archiwum transportowe pozostaje poza zakresem v1.

## Bajty podpisu

`manifest.json` zawiera jeden obiekt JSON, od 1 do 1 048 576 bajtów. `manifest.sig` zawiera dokładnie 64 surowe bajty podpisu, bez hex/base64 i bez końcowego LF.

Wiadomość Ed25519:

```text
ASCII("OENTIKE-PACK-MANIFEST-V1") || 0x00 || dokładne_bajty_manifest.json
```

Używamy zwykłego Ed25519 (bez Ed25519ph, bez wstępnego SHA-256 manifestu i bez parametru context). Zmiana białych znaków również wymaga nowego podpisu. Odbiorca weryfikuje oryginalne bajty przed parsowaniem; nie serializuje JSON ponownie. Generator zapisuje zwarty JSON bez końcowego LF, w kolejności pól pokazanej poniżej. Odbiorca akceptuje inne kolejności pól i białe znaki, jeśli podpis obejmuje właśnie te bajty.

Klucz publiczny (32 bajty) jest zaufanym wejściem aplikacji, a nie częścią paczki. V1 sprawdza jeden klucz: wskazany jawnie w CLI Go lub wbudowany podczas kompilacji Tauri. Identyfikatory kluczy i rotacja pozostają do integracji. Publiczny klucz dołączony przez nadawcę sam w sobie nie ustanawia zaufania.

## Schemat

```json
{
  "schema_version": 1,
  "area_id": "nadl-05-31",
  "release": 1,
  "created_at": "2026-10-02T12:00:00Z",
  "files": [
    {
      "path": "data/snapshot.json",
      "size": 6,
      "sha256": "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
      "format": "test-text/1",
      "source": "synthetic:test",
      "license": "CC0-1.0"
    }
  ]
}
```

Przykład jest syntetycznym wektorem protokołu: jego zasób zawiera sześć bajtów `hello` + LF, a nie rzeczywisty snapshot pogodowy. Rozszerzenie pliku nie steruje walidacją formatu.

Wszystkie pola są wymagane. Nieznane pola, inne wielkości liter w nazwach, duplikaty (również po dekodowaniu escape'ów), `null`, booleany i nadmiarowy JSON są odrzucane. Liczby muszą mieć zapis całkowity bez minusa, ułamka i wykładnika. W dniu 2026-10-03 doprecyzowano odrzucenie `-0` również w Go, aby wyeliminować różnicę interpretacji między bibliotekami. Obiekty nie mogą być zastępowane tablicami pozycyjnymi. Wartości tekstowe po dekodowaniu zawierają tylko drukowalne ASCII (0x20–0x7e); nie jest to ograniczenie treści zasobów. Polskie opisy pozostają w plikach atlasu/snapshotów. Adresy źródeł wymagające innych znaków zapisujemy jako URI z kodowaniem procentowym.

| Pole | Ograniczenie |
|---|---|
| `schema_version` | Dokładnie 1 |
| `area_id` | 1–64 znaki `[a-z0-9-]`, bez `-` na początku i końcu |
| `release` | 1–9 007 199 254 740 991; rosnące wydania w obrębie obszaru |
| `created_at` | Poprawny czas UTC, dokładnie `YYYY-MM-DDTHH:MM:SSZ` |
| `files` | 1–4096 pozycji, ścieżki unikalne, rosnący porządek bajtowy ASCII |
| `size` | 0–8 GiB na plik, suma do 32 GiB |
| `sha256` | 64 małe znaki hex, hash dokładnych bajtów zasobu |
| `format` | 1–64 znaki; identyfikator z wersją, np. `oent/1` |
| `source` | 1–2048 znaków; identyfikator/URI pochodzenia |
| `license` | 1–256 znaków; identyfikator/URI licencji |

`format`, `source` i `license` podlegają obecnie kontroli długości i alfabetu. Weryfikator nie rozpoznaje formatów ani nie ocenia praw do dystrybucji. Snapshot pogody musi we własnym wersjonowanym schemacie przechowywać `fetched_at` i zakres dat; `created_at` manifestu nie zastępuje tych pól.

Ścieżki są względne, długości do 240 bajtów, z maksymalnie 8 segmentami po 1–64 znaki `[a-z0-9._-]`. Segment nie może zaczynać się ani kończyć kropką. Zakazane są nazwy urządzeń Windows `con`, `prn`, `aux`, `nul`, `com0`–`com9`, `lpt0`–`lpt9`, także z rozszerzeniem. `manifest.json` i `manifest.sig` są zarezerwowane w korzeniu. Odrzucamy kolizje plik/katalog. Małe ASCII usuwa problem kolizji wielkości liter i normalizacji Unicode między platformami.

## Weryfikacja plików i granice zaufania

Kolejność: limit rozmiaru manifestu/podpisu → Ed25519 ze znanym kluczem → ścisły JSON i schemat → inwentarz katalogu → rozmiary i SHA-256. Każdy plik musi być wymieniony, a każdy katalog musi być rodzicem wymienionego pliku. Dodatkowe pliki, puste niewymienione katalogi, symlinki, FIFO i inne pliki specjalne są odrzucane. Katalogi czytamy porcjami po 64 wpisy. Dostęp do zasobów jest ograniczony przez `os.Root`.

Zasoby są hashowane kolejno z mmap read-only na Linux/macOS; nie kopiujemy całego zasobu do sterty Go. Puste pliki nie wymagają mapowania. Inne platformy zwracają jawny błąd hashowania. Obsługa platformy w generatorze Go nie przesądza platform docelowego klienta Rust.

Wywołujący musi zapewnić prywatny, niezmienny katalog roboczy przez cały czas weryfikacji i późniejszego użycia danych. Ta biblioteka nie zabezpiecza współbieżnych modyfikacji przez lokalnego autora plików. Skrócenie pliku używanego przez mmap może spowodować błąd procesu. Importer powinien tworzyć własne zwykłe pliki, bez symlinków/hardlinków z archiwum, oraz publikować je atomowo po wszystkich kontrolach. `VerifyFiles` nie utrzymuje otwartych uchwytów dla przyszłego czytnika.

Poprawny podpis nie oznacza poprawnej geometrii ani świeżej pogody. OENT/FlatBuffers/PMTiles i inne formaty nadal wymagają własnych walidatorów. CLI Go nie aktywuje wydania ani nie przechowuje historii importów. Importer Rust sprawdza wybrany `area_id` i wymaga numeru wydania większego od zapisanego w aktywnym stanie tego obszaru.

## CLI

Z katalogu `oentike-api`:

```sh
go run ./cmd/oentike-pack sign \
  -root /path/to/immutable-payload \
  -manifest /path/to/template.json \
  -seed-file /private/path/signing-seed.hex \
  -out /path/to/new-metadata-directory

go run ./cmd/oentike-pack verify \
  -root /path/to/immutable-payload \
  -manifest /path/to/new-metadata-directory/manifest.json \
  -signature /path/to/new-metadata-directory/manifest.sig \
  -public-key-file /trusted/path/public-key.hex
```

Szablon ma ten sam schemat; `size` może wynosić 0, a `sha256` zawierać 64 zera. Generator przelicza oba pola. Szablon, klucze i wynikowy katalog metadanych znajdują się poza katalogiem zasobów. Klucz prywatny jest 32-bajtowym seedem Ed25519 zapisanym jako 64 cyfry hex; plik publiczny zawiera 32-bajtowy klucz w tym samym kodowaniu. Dopuszczamy końcowy LF/CRLF. Klucze produkcyjne tworzymy i przechowujemy poza repozytorium, z ograniczonymi uprawnieniami dostępu. CLI nie generuje kluczy ani nie wypisuje ich treści.

`sign` tworzy nowy katalog i odmawia nadpisania istniejącego. Błąd zapisu może pozostawić niekompletne metadane; nie są publikowane ani aktywowane przez to narzędzie. Ta operacja nie zastępuje atomowego importera i nie zapewnia trwałości po utracie zasilania.

## Wektor i testy

[Stały wektor](../oentike-api/internal/pack/testdata/vector.json) zawiera dokładny manifest, podpis, publiczny klucz, jawny **testowy** seed i bajty zasobu w hex. Klucz testowy nie może wejść do produkcyjnego zbioru zaufanych kluczy; konfiguracja Tauri odrzuca go jawnie. Wektor jest weryfikowany w Go i Rust; test Rust dodatkowo odtwarza identyczne bajty podpisu i manifestu. Nie oznacza to równoważności bibliotek kryptograficznych dla wszystkich niekanonicznych lub słabych kluczy; Tauri stosuje `verify_strict` i odrzuca słaby klucz konfiguracyjny.

```sh
go test -race ./...
go test ./internal/pack -run '^$' -fuzz FuzzParse -fuzztime=10s
go test ./internal/pack -run '^$' -bench . -benchmem -count=3
```

Testy obejmują m.in. podmieniony klucz/podpis/manifest/zasób, prawidłowo podpisany błędny JSON, traversal, kolizje ścieżek, symlinki, FIFO, limity rozmiarów, brakujące/dodatkowe pliki oraz odmowę nadpisania wyników CLI. Wyniki pomiarów i ich zakres są w [PERFORMANCE.md](PERFORMANCE.md).
