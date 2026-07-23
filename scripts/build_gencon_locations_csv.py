#!/usr/bin/env python3
"""
Combines JSON files from location_data/ into a compact, deduplicated CSV file
containing stripped-down location info for database import and map matching.
Output: data/gencon_2026_locations.csv
"""

import csv
import json
import os
import re
import sys

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LOC_DIR = os.path.join(BASE_DIR, "location_data")
OUTPUT_CSV = os.path.join(BASE_DIR, "data", "gencon_2026_locations.csv")

ALLOWED_CATEGORIES = {"Spaces", "Events", "Hotels"}

def main():
    if not os.path.exists(LOC_DIR):
        print(f"Error: Directory {LOC_DIR} does not exist.", file=sys.stderr)
        sys.exit(1)

    geo_elements_by_id = {}
    total_raw_records = 0

    files = [f for f in os.listdir(LOC_DIR) if os.path.isfile(os.path.join(LOC_DIR, f))]
    print(f"Processing {len(files)} files in {LOC_DIR}...")

    for fname in sorted(files):
        fpath = os.path.join(LOC_DIR, fname)
        try:
            with open(fpath, "r", encoding="utf-8") as f:
                items = json.load(f)
                total_raw_records += len(items)
                for item in items:
                    src = item.get("_source", {})
                    scat = src.get("searchable_category")
                    
                    if scat not in ALLOWED_CATEGORIES:
                        continue
                    
                    eid = src.get("id")
                    if eid is None:
                        continue

                    # Deduplicate by unique id
                    if str(eid) not in geo_elements_by_id:
                        loc_label = src.get("map_feature", {}).get("properties", {}).get("location-label") or ""
                        geo_elements_by_id[str(eid)] = {
                            "id": eid,
                            "searchable_name": src.get("searchable_name", "") or "",
                            "location_label": loc_label,
                            "map_location": src.get("map_location", "") or "",
                            "category": scat,
                            "convention_id": src.get("convention_id", 0)
                        }
        except Exception as err:
            print(f"Warning: Failed to parse {fname}: {err}", file=sys.stderr)

    os.makedirs(os.path.dirname(OUTPUT_CSV), exist_ok=True)

    fieldnames = ["id", "searchable_name", "location_label", "map_location", "category", "convention_id"]
    
    sorted_records = sorted(geo_elements_by_id.values(), key=lambda r: (r["category"], r["id"]))

    with open(OUTPUT_CSV, "w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(sorted_records)

    file_size_kb = os.path.getsize(OUTPUT_CSV) / 1024.0

    print("\nExtraction & CSV Build Complete!")
    print(f"  Raw JSON Records Scanned: {total_raw_records}")
    print(f"  Deduplicated Map Location Pins: {len(sorted_records)}")
    print(f"  Output CSV File: {OUTPUT_CSV}")
    print(f"  File Size: {file_size_kb:.1f} KB")

if __name__ == "__main__":
    main()
