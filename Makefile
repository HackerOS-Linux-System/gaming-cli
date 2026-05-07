PREFIX      ?= /usr
BINDIR      := $(PREFIX)/bin
SYSTEMDDIR  := /usr/lib/systemd/system
DESKTOPDIR  := /usr/share/applications
SKELDESKTOP := /etc/skel/Desktop
ETCDIR      := /etc/hackeros
XDGDIR      := /etc/xdg
STATEDIR    := /var/lib/hackeros-gaming
LOGDIR      := /var/log/hackeros-gaming

SRCDIR      := source-code
OUTDIR      := bin

.PHONY: all build deps install uninstall clean

all: build

# ── Zależności ────────────────────────────────────────────────────────────────
# go mod tidy pobiera zależności i generuje go.sum — wymagane przed pierwszym buildem.

deps:
	@echo ">>> Pobieranie zależności i generowanie go.sum..."
	cd $(SRCDIR)/gaming-cli        && go mod tidy && go mod download
	cd $(SRCDIR)/gaming            && go mod tidy && go mod download
	cd $(SRCDIR)/gamescope-manager && go mod tidy && go mod download
	@echo "    Zależności gotowe."
	@echo ""

# ── Budowanie ────────────────────────────────────────────────────────────────

build: deps
	@echo ">>> [1/3] Budowanie gaming-cli..."
	@mkdir -p $(OUTDIR)
	cd $(SRCDIR)/gaming-cli        && go build -ldflags="-s -w" -o ../../$(OUTDIR)/gaming-cli .
	@echo ">>> [2/3] Budowanie gaming (wrapper)..."
	cd $(SRCDIR)/gaming            && go build -ldflags="-s -w" -o ../../$(OUTDIR)/gaming .
	@echo ">>> [3/3] Budowanie gamescope-manager..."
	cd $(SRCDIR)/gamescope-manager && go build -ldflags="-s -w" -o ../../$(OUTDIR)/gamescope-manager .
	@echo ""
	@echo "    Gotowe! Binaria: $(OUTDIR)/"
	@ls -lh $(OUTDIR)/

# ── Instalacja ────────────────────────────────────────────────────────────────

install: build
	@echo ""
	@echo ">>> Instalacja HackerOS Gaming Edition v0.0.1"
	@echo ""

	@echo "  [bin] Instalacja binariów..."
	install -Dm755 $(OUTDIR)/gaming-cli        $(DESTDIR)$(BINDIR)/gaming-cli
	install -Dm755 $(OUTDIR)/gaming            $(DESTDIR)$(BINDIR)/gaming
	install -Dm755 $(OUTDIR)/gamescope-manager $(DESTDIR)$(BINDIR)/gamescope-manager

	@echo "  [systemd] Instalacja serwisu..."
	install -Dm644 systemd/hackeros-game-mode.service \
		$(DESTDIR)$(SYSTEMDDIR)/hackeros-game-mode.service

	@echo "  [desktop] Instalacja skrótu systemowego..."
	install -Dm644 dist/hackeros-game-mode.desktop \
		$(DESTDIR)$(DESKTOPDIR)/hackeros-game-mode.desktop

	@echo "  [skel] Instalacja skrótu do /etc/skel/Desktop/ ..."
	install -dm755 $(DESTDIR)$(SKELDESKTOP)
	install -Dm644 dist/hackeros-game-mode.desktop \
		$(DESTDIR)$(SKELDESKTOP)/hackeros-game-mode.desktop

	@echo "  [config] Instalacja konfiguracji .hk..."
	install -dm755 $(DESTDIR)$(ETCDIR)
	if [ ! -f $(DESTDIR)$(ETCDIR)/gamescope-manager.hk ]; then \
		install -Dm644 dist/gamescope-manager.hk \
			$(DESTDIR)$(ETCDIR)/gamescope-manager.hk; \
	fi

	@echo "  [marker] Instalacja znacznika Gaming Edition..."
	install -Dm644 dist/hackeros-gaming-edition \
		$(DESTDIR)/etc/hackeros-gaming-edition

	@echo "  [kcm] Instalacja kcm-about-distrorc..."
	install -dm755 $(DESTDIR)$(XDGDIR)
	install -Dm644 dist/kcm-about-distrorc \
		$(DESTDIR)$(XDGDIR)/kcm-about-distrorc

	@echo "  [dirs] Tworzenie katalogów stanu i logów..."
	install -dm755 $(DESTDIR)$(STATEDIR)
	install -dm755 $(DESTDIR)$(LOGDIR)

	@echo ""
	@echo "    Instalacja zakończona!"
	@echo ""
	@echo "    Następne kroki:"
	@echo "      sudo systemctl daemon-reload"
	@echo "      sudo systemctl enable hackeros-game-mode.service"
	@echo ""
	@echo "    Użycie:"
	@echo "      gaming-cli          # otwórz TUI"
	@echo "      gaming game         # przełącz na tryb gry"
	@echo "      gaming desktop      # przełącz na pulpit"
	@echo ""

# ── Odinstalowywanie ──────────────────────────────────────────────────────────

uninstall:
	@echo ">>> Odinstalowywanie HackerOS Gaming Edition..."
	rm -f $(DESTDIR)$(BINDIR)/gaming-cli
	rm -f $(DESTDIR)$(BINDIR)/gaming
	rm -f $(DESTDIR)$(BINDIR)/gamescope-manager
	rm -f $(DESTDIR)$(SYSTEMDDIR)/hackeros-game-mode.service
	rm -f $(DESTDIR)$(DESKTOPDIR)/hackeros-game-mode.desktop
	rm -f $(DESTDIR)$(SKELDESKTOP)/hackeros-game-mode.desktop
	rm -f $(DESTDIR)/etc/hackeros-gaming-edition
	@echo ">>> Odinstalowano."
	@echo "    Uwaga: /etc/hackeros/ i /var/lib/hackeros-gaming/ pozostają."

# ── Czyszczenie ───────────────────────────────────────────────────────────────

clean:
	rm -rf $(OUTDIR)/
	@echo ">>> Wyczyszczono."
