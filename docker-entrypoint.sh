#!/usr/bin/env bash

glide --debug install
go-wrapper install
exec $@
