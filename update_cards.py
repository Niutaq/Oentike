import json
import os
from datetime import datetime

cards_dir = "oentike-atlas/packs/pilot-0.0.1/cards"
target_cards = [
    "boletus-edulis.json",
    "boletus-reticulatus.json",
    "cantharellus-cibarius.json",
    "laetiporus-sulphureus.json",
    "macrolepiota-procera.json",
    "suillus-luteus.json",
    "tylopilus-felleus.json"
]

for card_name in target_cards:
    filepath = os.path.join(cards_dir, card_name)
    with open(filepath, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    sci_name = data["scientific_name"]
    # URL encode scientific name (e.g. Boletus edulis -> Boletus%20edulis)
    sci_name_encoded = sci_name.replace(" ", "%20")

    data["citations"] = [
        {
            "label": "MycoBank",
            "url": f"https://www.mycobank.org/page/Name%20details%20page/name/{sci_name_encoded}"
        },
        {
            "label": "Index Fungorum",
            "url": f"http://www.indexfungorum.org/names/names.asp?strSort={sci_name_encoded}"
        }
    ]

    data["reviewed_at"] = "2026-09-24"

    with open(filepath, 'w', encoding='utf-8') as f:
        json.dump(data, f, indent=2, ensure_ascii=False)
        f.write("\n")

print("Cards updated!")
