# Atlas Oentike

Robocza paczka zawiera **11 gatunków**. `packs/pilot-0.0.1/pack.json` jest źródłem listy kart wyświetlanych przez UI; dodanie karty do manifestu tworzy jej stronę bez ręcznego importu w TypeScript.

## Sprawdzanie

Z katalogu głównego:

```sh
./oentike atlas
```

Walidator korzysta z JSON Schema 2020-12 i sprawdza manifest, wersje, odnośniki do podobnych gatunków, obecność SVG i zgodność obu kopii ilustracji. Jest również uruchamiany przed buildem UI. Brak źródeł w starszych kartach jest raportowany jako zaległość redakcyjna; błędy struktury blokują build.

## Dodanie gatunku

1. Dodaj JSON zgodny z `schema/card.schema.json` do `packs/pilot-0.0.1/cards/`.
2. Dodaj jego ścieżkę do `pack.json`. Nazwa polska/angielska jest główna, łacińska pomocnicza.
3. Pisz własne, krótkie opisy na podstawie podanych cytacji. Wskazuj region, którego dotyczy sezonowość. Nie wyprowadzaj statusu prawnego z braku wzmianki w źródle.
4. Sprawdź, czy każdy `lookalikes[].slug` wskazuje istniejącą kartę. `folds` oznacza fałdy pieprznika, `gills` blaszki, `pores` pory.
5. Umieść autorskie SVG w `atlas/art/v0/` w katalogu głównym repo oraz identyczną kopię w `oentike-web/public/atlas/art/v0/`. Ścieżki `art` są absolutnymi ścieżkami zasobów UI.
6. Uruchom walidację i build. Obejrzyj render SVG; zgodność XML nie gwarantuje poprawności morfologicznej.

## Stan redakcyjny

Cztery nowe karty (maślak ziarnisty, lisówka pomarańczowa, czubajka czerwieniejąca i muchomor zielonawy) mają cytowane opisy i schematyczny portret w kolorach Oentike. Status ochrony i niezweryfikowane identyfikatory są null. Ilustracje mają barwy umowne i nie zastępują klucza do oznaczania gatunków.

Siedem wcześniejszych kart wymaga dodania źródeł i przeglądu także istniejących stwierdzeń o ochronie. Żadna karta nie jest jeszcze oznaczona jako przejrzana przez człowieka. Pole `reviewed_at` ustawia człowiek po kontroli treści i cytacji, nie automat po pobraniu strony.

Paczka jest robocza i niepodpisana. Przed publikacją należy nadać nową, niezmienną wersję i uzgodnić licencję treści i ilustracji. Nie kopiujemy opisów ani fotografii z atlasów zewnętrznych.
