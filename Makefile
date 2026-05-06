# Copyright (c) 2026 HackerOS Team
# SPDX-License-Identifier: BSD-3-Clause

PREFIX      ?= /usr
BINDIR      := $(PREFIX)/bin
SYSTEMD_DIR := /usr/lib/systemd/system

.PHONY: all build install uninstall clean

all: build

build:
	@echo ">>> Budowanie gaming-cli..."
	cd gaming-cli && go build -o ../bin/gaming-cli .
	@echo ">>> Budowanie gaming (wrapper)..."
	cd gaming && go build -o ../bin/gaming .
	@echo ">>> Budowanie gamescope-manager..."
	cd gamescope-manager && go build -o ../bin/gamescope-manager .
	@echo ">>> Gotowe: bin/"

install: build
	@echo ">>> Instalacja binariów do $(BINDIR)..."
	install -Dm755 bin/gaming-cli        $(DESTDIR)$(BINDIR)/gaming-cli
	install -Dm755 bin/gaming            $(DESTDIR)$(BINDIR)/gaming
	install -Dm755 bin/gamescope-manager $(DESTDIR)$(BINDIR)/gamescope-manager

	@echo ">>> Instalacja serwisów systemd..."
	install -Dm644 systemd/hackeros-game-mode.service \
		$(DESTDIR)$(SYSTEMD_DIR)/hackeros-game-mode.service

	@echo ">>> Tworzenie katalogów stanu..."
	install -dm755 $(DESTDIR)/var/lib/hackeros-gaming
	install -dm755 $(DESTDIR)/var/log/hackeros-gaming
	install -dm755 $(DESTDIR)/etc/hackeros

	@echo ">>> Instalacja znacznika edycji Gaming..."
	install -Dm644 dist/hackeros-gaming-edition \
		$(DESTDIR)/etc/hackeros-gaming-edition

	@echo ">>> Instalacja kcm-about-distrorc..."
	install -Dm644 dist/kcm-about-distrorc \
		$(DESTDIR)/etc/xdg/kcm-about-distrorc

	@echo ""
	@echo "    Instalacja zakończona."
	@echo "    Przeładuj systemd: sudo systemctl daemon-reload"

uninstall:
	rm -f  $(DESTDIR)$(BINDIR)/gaming-cli
	rm -f  $(DESTDIR)$(BINDIR)/gaming
	rm -f  $(DESTDIR)$(BINDIR)/gamescope-manager
	rm -f  $(DESTDIR)$(SYSTEMD_DIR)/hackeros-game-mode.service
	rm -f  $(DESTDIR)/etc/hackeros-gaming-edition
	@echo ">>> Odinstalowano."

clean:
	rm -rf bin/
	@echo ">>> Wyczyszczono."
