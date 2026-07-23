#!/usr/bin/env python3
"""
Imports data/gencon_2026_locations.csv into PostgreSQL table public.gencon_locations.
"""

import csv
import os
import sys

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CSV_PATH = os.path.join(BASE_DIR, "data", "gencon_2026_locations.csv")
SQL_MIGRATION_PATH = os.path.join(BASE_DIR, "internal", "postgres", "gencon_locations.sql")

def generate_insert_sql(csv_path):
    inserts = []
    with open(csv_path, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            eid = int(row["id"])
            sname = row["searchable_name"].replace("'", "''")
            ll_val = row["location_label"].replace("'", "''")
            llabel = f"'{ll_val}'" if row["location_label"] else "NULL"
            mlocation = row["map_location"].replace("'", "''")
            cat = row["category"].replace("'", "''")
            cid = int(row["convention_id"]) if row["convention_id"] else 0

            sql = f"({eid}, '{sname}', {llabel}, '{mlocation}', '{cat}', {cid})"
            inserts.append(sql)
    return inserts

def main():
    if not os.path.exists(CSV_PATH):
        print(f"Error: CSV file {CSV_PATH} not found.", file=sys.stderr)
        sys.exit(1)

    inserts = generate_insert_sql(CSV_PATH)
    
    # Generate seed SQL file
    seed_sql_path = os.path.join(BASE_DIR, "data", "seed_gencon_locations.sql")
    with open(seed_sql_path, "w", encoding="utf-8") as f:
        f.write("-- Ensure table exists\n")
        with open(SQL_MIGRATION_PATH, "r", encoding="utf-8") as mig:
            f.write(mig.read())
        f.write("\n\n-- Seed location pins\n")
        f.write("INSERT INTO public.gencon_locations (id, searchable_name, location_label, map_location, category, convention_id) VALUES\n")
        f.write(",\n".join(inserts))
        f.write("\nON CONFLICT (id) DO UPDATE SET\n")
        f.write("  searchable_name = EXCLUDED.searchable_name,\n")
        f.write("  location_label = EXCLUDED.location_label,\n")
        f.write("  map_location = EXCLUDED.map_location,\n")
        f.write("  category = EXCLUDED.category,\n")
        f.write("  convention_id = EXCLUDED.convention_id;\n")

    print(f"Successfully generated seed SQL script: {seed_sql_path}")
    print(f"Contains {len(inserts)} location pin UPSERT statements.")

if __name__ == "__main__":
    main()
