# Lokalny importer Rust/Tauri

Stan 2026-10-03: natywne komendy Tauri, bez nowego ekranu importu. Zgodny z [manifestem v1](PACK-FORMAT.md). Przyjmuje istniejący katalog zasobów, osobny manifest i podpis z CLI Go. Nie pobiera danych, nie rozpakowuje archiwów i nie uruchamia API.

## Zaufanie i wywołanie

Publiczny klucz Ed25519 należy ustawić jako `OENTIKE_PACK_PUBLIC_KEY_HEX` **podczas kompilacji** Tauri: dokładnie 64 małe cyfry hex. `build.rs` śledzi zmianę tej zmiennej. Brak klucza, niepoprawny klucz, słaby punkt krzywej lub klucz z publicznego wektora testowego blokuje import. Klucz prywatny pozostaje wyłącznie w generatorze; aplikacja go nie potrzebuje. W tej zmianie nie utworzono klucza produkcyjnego.

Komendy rejestrowane w `src-tauri/src/lib.rs`:

```ts
await invoke('import_offline_pack', {
  areaId: 'nadl-05-31',
  payloadDir: '/absolute/path/to/payload',
  manifestPath: '/absolute/path/to/metadata/manifest.json',
  signaturePath: '/absolute/path/to/metadata/manifest.sig',
});

await invoke('get_offline_pack', { areaId: 'nadl-05-31' });
```

Importer zwraca `{ pack: { areaId, release, createdAt, fileCount, totalBytes }, durabilityConfirmed }`. Odczyt zwraca taki sam obiekt `pack` lub `null`, jeśli obszar nie ma aktywnego wydania. Uszkodzony stan jest błędem, nie `null`. Komendy wykonują ciężkie I/O przez `spawn_blocking`. Frontend nie przekazuje klucza zaufania ani ścieżki docelowego magazynu.

Importer korzysta z Unixowych uchwytów katalogów, `openat` i blokad plikowych; został przetestowany na macOS/arm64. Linux i platformy mobilne wymagają osobnego uruchomienia testów. Windows zwraca jawny błąd nieobsługiwanej platformy.

## Przechowywanie i commit

Magazyn znajduje się w `app_data_dir()/offline-packs-v1`, z uprawnieniami `0700`:

```text
offline-packs-v1/
  import.lock
  nadl-05-31/
    active.json
    releases/
      pack-<losowy identyfikator>/
        manifest.json
        manifest.sig
        payload/...
```

1. Blokada `flock` ogranicza cały magazyn do jednego importu naraz, także między procesami. Inny importer otrzymuje błąd zajętości. Plik blokady nigdy nie jest podmieniany ani usuwany.
2. Ograniczony odczyt metadanych, Ed25519, ścisły JSON, powiązanie z wybranym obszarem i kontrola rosnącego wydania. Ponowny import tego samego numeru również jest odrzucany. Uszkodzony aktywny manifest/pointer blokuje aktualizację; nie zerujemy historii.
3. Kontrola pełnego inwentarza wejścia, rozmiarów i dostępnego miejsca: suma zadeklarowanych plików, manifest i 16 MiB rezerwy. To kontrola wstępna, nie rezerwacja dysku; późniejsze błędy zapisu nadal przerywają import.
4. Kopia do nowego `.stage-*` wewnątrz magazynu. Jeden bufor 64 KiB jest używany ponownie; zasoby mają własne zwykłe pliki. Odczyt każdego komponentu ścieżki odbywa się względem otwartego katalogu z `NOFOLLOW`; `NONBLOCK` zapobiega zawieszeniu na FIFO. Symlinki, hardlinki, pliki specjalne, brakujące i dodatkowe wpisy są odrzucane.
5. Po zamknięciu uchwytu zapisującego SHA-256 jest liczony z prywatnego pliku przez mmap. Pliki docelowe otrzymują `0400`. Zasoby wejściowe mogą się zmieniać; kopiowanie wykrywa rozmiar, a hash potwierdza faktycznie skopiowane bajty. Nie mapujemy niekontrolowanych plików źródłowych.
6. `sync_all` plików i katalogów, publikacja kompletnego katalogu wydania przez rename, przygotowanie i synchronizacja nowego wskaźnika. Atomowe rename małego `active.json` jest punktem aktywacji. Wszystkie te pliki znajdują się na tym samym systemie plików.

Przed rename wskaźnika każdy zwrócony błąd pozostawia poprzedni `active.json`. Po rename nowa paczka jest aktywna: jeżeli końcowy fsync katalogu zawiedzie, wynik nadal oznacza sukces aktywacji, ale `durabilityConfirmed` wynosi `false`. Nie wolno pokazywać tego użytkownikowi jako nieudanej aktywacji ani obiecywać trwałości po utracie zasilania. Nawet udane fsync nie jest testem zachowania sprzętu przy zaniku zasilania.

Odczyt aktywnego wydania ponownie sprawdza podpis, powiązanie wskaźnika z hashem manifestu, inwentarz i hashe zasobów. Trwałość stanu wynika z plików, bez pamięci podręcznej wymaganej do restartu. Starych wydań nie nadpisujemy ani nie usuwamy, więc ich otwarte mapowania nadal działają.

## Awarie i granice

| Sytuacja | Rezultat |
|---|---|
| Błąd przed publikacją katalogu wydania | Poprzednia paczka aktywna; RAII usuwa katalog roboczy bieżącej próby |
| Błąd po publikacji, przed commit | Poprzednia paczka aktywna; może pozostać kompletne osierocone wydanie |
| Przerwanie procesu | Blokada zwalnia się przy zamknięciu procesu; staging/osierocone wydania nie są wybierane jako aktywne |
| Ponowny start aplikacji | Wskaźnik jest odczytywany i weryfikowany z dysku; nie wybieramy automatycznie „najwyższego” katalogu |
| Starsze lub to samo wydanie | Odrzucenie przed kopiowaniem danych |
| Uszkodzone dane aktywnej paczki | Błąd weryfikacji; brak cichego fallbacku |

Sprzątanie starych/osieroconych wydań i katalogów po przerwaniu procesu jest osobnym zadaniem. Importer nie usuwa ich automatycznie; każde kolejne wydanie zużywa dodatkowe miejsce.

Granica zaufania obejmuje prywatny magazyn aplikacji. Proces z tym samym UID lub administrator może zmienić uprawnienia i pliki; importer nie chroni przed takim przeciwnikiem ani przed ręcznym usunięciem historii antyrollback. mmap w Rust wymaga jednego jawnego bloku `unsafe`, ograniczonego do prywatnych kopii i niezmiennych opublikowanych plików. To wymóg API [memmap2](https://docs.rs/memmap2/0.9.9/memmap2/struct.MmapOptions.html), wynikający z możliwości zmiany pliku poza procesem. [Weryfikacja Ed25519](https://docs.rs/ed25519-dalek/2.2.0/ed25519_dalek/struct.VerifyingKey.html#method.verify_strict) odbywa się przed parsowaniem podpisanego JSON.

Aktywacja oznacza autentyczny inwentarz i zgodne bajty zasobów. Nie oznacza poprawnej topologii, kompletności zasobów renderera ani aktualności pogody. Walidatory OENT/FlatBuffers/PMTiles, ekran wyboru paczki, użycie zasobów w rendererze, rotacja kluczy i archiwum transportowe pozostają kolejnymi zakresami.

## Sprawdzenie

Z `oentike-web/src-tauri`:

```sh
cargo test --locked --all-targets
cargo clippy --locked --all-targets -- -D warnings -A clippy::result_large_err
```

Wyjątek Clippy dotyczy dużego `tonic::Status` w istniejącym wygenerowanym kliencie gRPC. Nie zmieniono tego generatora ani kontraktu RPC.

Testy obejmują wspólny wektor Go/Rust, poprawnie podpisany błędny JSON, deterministyczne mutacje parsera, błędy hashy/podpisów, traversal, symlinki/hardlinki/FIFO, izolację kopii od źródła, aktualizację przy otwartym starym mmap, restart magazynu, antyrollback, blokadę współbieżności, niekompletny stan i wstrzyknięte błędy przed commit. Test ponownego otwarcia magazynu nie jest testem restartu GUI ani fizycznego zaniku zasilania.
