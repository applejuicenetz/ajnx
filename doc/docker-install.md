# Docker-Anleitung: aJnX

aJnX besteht aus zwei Diensten: dem **Gateway** (kümmert sich um die Verbindung zum appleJuice-Core) und dem **UI** (das Web-Interface, das du im Browser öffnest). Beide laufen als Docker-Container und kommunizieren intern miteinander.

---

## Voraussetzungen

- Docker (ab Version 20.x)
- Docker Compose (ab Version 2.x, also `docker compose` ohne Bindestrich)
- Ein laufender appleJuice-Core, den das Gateway erreichen kann

---

## Option 1: Prebuilt Images (empfohlen)

Du brauchst das Repository nicht klonen. Erstelle einfach eine `docker-compose.yaml` mit folgendem Inhalt:

```yaml
services:
  gateway:
    image: ghcr.io/applejuicenetz/ajnx-gateway:latest
    container_name: ajnx-gateway
    environment:
      - GATEWAY_PORT=${GATEWAY_PORT:-9859}
      - SETUP_LOCK_PATH=/data/setup.lock
    ports:
      - "${GATEWAY_PORT:-9859}:${GATEWAY_PORT:-9859}"
    user: "1000:1000"
    volumes:
      - ./data/ui:/data
    extra_hosts:
      - "host.docker.internal:host-gateway"
    restart: unless-stopped

  ui:
    image: ghcr.io/applejuicenetz/ajnx-ui:latest
    container_name: ajnx-ui
    environment:
      - UI_PORT=${UI_PORT:-9858}
      - UI_GATEWAY_URL=http://gateway:${GATEWAY_PORT:-9859}
      - SETUP_LOCK_PATH=/data/setup.lock
    ports:
      - "${UI_PORT:-9858}:${UI_PORT:-9858}"
    user: "1000:1000"
    depends_on:
      - gateway
    volumes:
      - ./data/ui:/data
    extra_hosts:
      - "host.docker.internal:host-gateway"
    restart: unless-stopped
```

Dann starten:

```bash
docker compose up -d
```

Docker lädt die Images automatisch von der GitHub Container Registry herunter.

---

## Option 2: Selbst bauen (aus dem Quellcode)

Falls du Änderungen am Code machen oder lokal testen willst:

```bash
git clone https://github.com/applejuicenetz/ajnx.git
cd ajnx
docker compose up -d --build
```

Das `--build` sorgt dafür, dass die Images frisch gebaut werden. Ohne `--build` nimmt Docker gecachte Layer, falls vorhanden.

---

## Setup nach dem ersten Start

Beim ersten Start läuft automatisch ein Setup-Assistent. Öffne dazu:

```
http://localhost:9858
```

Dort gibst du die Verbindungsdaten deines appleJuice-Cores ein. Nach dem Setup wird eine Lock-Datei unter `/data/setup.lock` angelegt, damit der Assistent beim nächsten Start nicht erneut erscheint.

---

## Ports anpassen

Standardmäßig läuft das UI auf Port `9858` und das Gateway auf Port `9859`. Du kannst beides über Umgebungsvariablen ändern, ohne die `docker-compose.yaml` anzufassen.

Erstelle dazu eine `.env`-Datei im selben Verzeichnis:

```env
UI_PORT=8080
GATEWAY_PORT=8081
```

Docker Compose liest diese Datei automatisch ein.

---

## Nützliche Befehle

Container stoppen:
```bash
docker compose down
```

Logs ansehen (beide Dienste):
```bash
docker compose logs -f
```

Logs nur für einen Dienst:
```bash
docker compose logs -f ui
docker compose logs -f gateway
```

Container-Status prüfen:
```bash
docker compose ps
```

Images auf die neueste Version aktualisieren (Option 1):
```bash
docker compose pull
docker compose up -d
```

---

## Datenspeicherung

Beide Container mounten `./data/ui` nach `/data`. Dort liegen die Setup-Lock-Datei und ggf. weitere Konfigurationsdaten. Das Verzeichnis wird beim ersten Start automatisch angelegt, sofern Docker die nötigen Schreibrechte hat.

Wenn du aJnX komplett zurücksetzen willst (z.B. um den Setup-Assistenten erneut auszuführen), lösche die Lock-Datei:

```bash
rm ./data/ui/setup.lock
```

Danach die Container neu starten:

```bash
docker compose restart
```

---

## Hinweis zum User

Beide Container laufen mit `user: "1000:1000"`. Das ist der Standard-UID auf den meisten Linux-Systemen. Falls das `./data/ui`-Verzeichnis einem anderen Benutzer gehört, kann es zu Berechtigungsfehlern kommen. In dem Fall:

```bash
mkdir -p ./data/ui
chown -R 1000:1000 ./data/ui
```

---

## Verbindung zum appleJuice-Core

Das Gateway muss den Core erreichen können. Läuft der Core direkt auf dem Host (also nicht in Docker), nutze die Adresse `host.docker.internal` im Setup-Assistenten. Der entsprechende Eintrag in der `docker-compose.yaml` (`extra_hosts`) sorgt dafür, dass dieser Hostname im Container auflösbar ist.

Läuft der Core selbst in Docker, kannst du ihn stattdessen über seinen Container-Namen oder eine gemeinsame Docker-Network-Konfiguration ansprechen.