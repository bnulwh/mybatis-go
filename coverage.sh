#!/usr/bin/env bash
go test -v ./... -coverprofile=cover.out
gocov convert cover.out |gocov-html > coverage.html
