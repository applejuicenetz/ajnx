# appleJuice phpGUI

![](https://img.shields.io/github/release/applejuicenetz/ajnx.svg)
![](https://img.shields.io/github/downloads/applejuicenetz/ajnx/total)
![](https://img.shields.io/github/license/applejuicenetz/ajnx.svg)

![](https://github.com/applejuicenetz/ajnx/actions/workflows/container.yml/badge.svg)
![](https://img.shields.io/docker/pulls/applejuicenetz/ajnx)
![](https://img.shields.io/docker/image-size/applejuicenetz/ajnx)

# AJNX – appleJuice Next eXperience

AJNX ist der moderne Nachfolger der legendären PHP-GUI für das appleJuice-Netzwerk. Während die alte PHP-GUI über Jahre gute Dienste geleistet hat, wurde es Zeit für etwas Frisches, Schnelles und Einfaches.

Dieses Projekt trennt die Logik sauber in ein **Gateway** (Go) und ein **Frontend** (Go + HTMX). Kein Apache, kein PHP-Frust – einfach ein Docker-Container und los geht's.

## Features

* **Echtzeit-Dashboard:** Alle wichtigen Stats (Speed, Credits, Server-Status) auf einen Blick, ohne die Seite ständig neu laden zu müssen.
* **Modernes Web-Interface:** Komplett responsiv. Funktioniert auf dem Desktop genauso gut wie auf dem Smartphone.
* **Einfaches Setup:** Ein Wizard führt dich beim ersten Start durch die Verbindung zum Core.
* **AJFSP-Protokoll-Handler:** Klick auf einen appleJuice-Link im Browser, und AJNX übernimmt den Rest.
* **Leichtgewichtig:** Minimaler Ressourcenverbrauch dank Go.

## Schnellstart mit Docker

Der einfachste Weg AJNX zu nutzen, ist über Docker Compose.

1. Repository klonen:
   ```bash
   git clone https://github.com/appleJuiceNetz/ajnx.git
   cd ajnx
   ```

2. Container starten:
   ```bash
   docker compose up -d --build
   ```

3. AJNX aufrufen:
   Öffne `http://localhost:9858` in deinem Browser und folge dem Setup-Assistenten.

## Warum AJNX?

Die alte PHP-GUI war großartig, aber sie ist technisch in die Jahre gekommen. AJNX setzt auf aktuelle Technologien wie **Go** für das Backend und **HTMX** für die Interaktivität. Das bedeutet: weniger Overhead, keine komplizierten Webserver-Konfigurationen und eine Benutzererfahrung, die sich wie eine echte App anfühlt.

## Entwicklung

AJNX ist von der Community für die Community. Wenn du Ideen hast oder Fehler findest, mach ein Issue auf oder schick einen Pull-Request.

### Anforderungen (für lokale Entwicklung)
* Go 1.22+
* [Templ](https://templ.guide/)
* Docker

---
**Hinweis:** AJNX ist ein Community-Projekt und steht in keiner offiziellen Verbindung zu den ursprünglichen Entwicklern des appleJuice-Cores. Nutzung auf eigene Gefahr.
