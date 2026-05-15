#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <event_id_1> <event_id_2>"
    exit 1
fi

ID1=$1
ID2=$2

cat <<EOF
WITH ids AS (
    SELECT '$ID1' AS id1, '$ID2' AS id2
),
e1 AS (SELECT * FROM events, ids WHERE event_id = id1),
e2 AS (SELECT * FROM events, ids WHERE event_id = id2)
SELECT 
    'title' AS field, 
    e1.title AS val1, 
    e2.title AS val2,
    (e1.title = e2.title) AS matches
FROM e1, e2
UNION ALL
SELECT 
    'short_description', 
    e1.short_description, 
    e2.short_description,
    (e1.short_description = e2.short_description)
FROM e1, e2
UNION ALL
SELECT 
    'short_category', 
    e1.short_category, 
    e2.short_category,
    (e1.short_category = e2.short_category)
FROM e1, e2
UNION ALL
SELECT 
    'active', 
    e1.active::text, 
    e2.active::text,
    (e1.active = e2.active)
FROM e1, e2;
EOF
