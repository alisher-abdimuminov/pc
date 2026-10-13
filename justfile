# Thin wrappers around the Makefile (which wraps docker compose).
default:
    @just --list

up:
    @make up

down:
    @make down

logs:
    @make logs

ps:
    @make ps

migrate:
    @make migrate

createadmin:
    @make createadmin

seed:
    @make seed

sqlc:
    @make sqlc

test:
    @make test

test-go:
    @make test-go

test-face:
    @make test-face

test-web:
    @make test-web

web-dev:
    @make web-dev

resetpassword:
    @make resetpassword

seeddemo:
    @make seeddemo
