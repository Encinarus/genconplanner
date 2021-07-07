#!/bin/sh

go build -o bin/bgg github.com/Encinarus/genconplanner/cmd/bgg && \
go build -o bin/update github.com/Encinarus/genconplanner/cmd/update && \
go build -o bin/web github.com/Encinarus/genconplanner/cmd/web