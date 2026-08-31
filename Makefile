# The datastores every test suite reads live in a bucket, and go test runs each
# package as its own process: ten of them read AllPrintings, so a plain
# `go test ./...` fetched and unpacked the same file ten times, all at once and
# all competing for the connection. Unpacking them here first, once, takes the
# suite from about nine minutes to under three.
#
# Nothing in the tree knows about this. The suites read the paths their
# environment gives them, and `make test` gives them local ones.

DATASTORES ?= $(HOME)/.cache/mtgban-datastore

# How old an unpacked copy may be before it is fetched again, in minutes. The
# datastores publish nightly, so an hour keeps a run from ever reasoning about
# yesterday's file while covering a suite run many times over.
#
# The copy is written through `xz -dc` rather than unpacked in place, so its
# timestamp is when it was fetched. `xz -d` restores the one the archive
# carries, which is the datastore's own build time and always older than the
# window - every run would fetch again.
DATASTORE_TTL ?= 60

# game/basename, as the bucket lays them out.
DATASTORE_FILES := \
	magic/AllPrintings \
	lorcana/lorcana \
	riftbound/riftbound \
	onepiece/onepiece \
	pokemon/pokemon \
	yugioh/yugioh \
	fleshandblood/fleshandblood

DATASTORE_ENV := \
	ALLPRINTINGS5_PATH=$(DATASTORES)/AllPrintings.json \
	LORCANA_PATH=$(DATASTORES)/lorcana.json \
	RIFTBOUND_PATH=$(DATASTORES)/riftbound.json \
	ONEPIECE_PATH=$(DATASTORES)/onepiece.json \
	POKEMON_PATH=$(DATASTORES)/pokemon.json \
	YUGIOH_PATH=$(DATASTORES)/yugioh.json \
	FLESHANDBLOOD_PATH=$(DATASTORES)/fleshandblood.json

.PHONY: datastores test test-remote vet lint clean-datastores help

help: ## List the targets
	@grep -hE '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*## /\t/' | expand -t 22

datastores: ## Unpack the published datastores, skipping what is already fresh
	@mkdir -p $(DATASTORES)
	@set -e; for pair in $(DATASTORE_FILES); do \
		out="$(DATASTORES)/$${pair##*/}.json"; \
		if [ -f "$$out" ] && [ -z "$$(find "$$out" -mmin +$(DATASTORE_TTL))" ]; then \
			continue; \
		fi; \
		echo "fetching $$pair"; \
		b2 file download --no-progress "b2://mtgban-datastore/$$pair.json.xz" "$$out.xz"; \
		xz -dc "$$out.xz" > "$$out"; \
		rm -f "$$out.xz"; \
	done

test: datastores ## Run every suite against the unpacked datastores
	@$(DATASTORE_ENV) go test ./...

test-remote: ## Run every suite against the bucket, as CI's paths are configured
	@go test ./...

vet: ## go vet, with the datastores in place for the tests it type-checks
	@go vet ./...

clean-datastores: ## Drop the unpacked copies
	@rm -rf $(DATASTORES)
