# BGE-M3 CPU Deployment

This directory deploys the `BAAI/bge-m3` embedding model with the CPU image of Hugging Face Text Embeddings Inference (TEI). The endpoint binds to `127.0.0.1:8081` by default, so it is reachable only from the local machine.

Download the model before starting TEI. TEI then loads `/data/bge-m3` from the mounted local directory instead of downloading the weights itself. This avoids the unreliable large-file fallback download and preserves the model across container recreation.

## Prerequisites

- Docker Engine or Docker Desktop with Docker Compose v2
- Either the Hugging Face CLI (`hf`) or Git LFS
- A Docker network named `traefik`

Create the external network once when it does not already exist:

```bash
docker network inspect traefik >/dev/null 2>&1 || docker network create traefik
```

## Download The Model

Run the following commands from this directory. Choose one download method only; do not run both against `data/tei/cache/bge-m3` at the same time.

### Hugging Face CLI

```bash
hf download BAAI/bge-m3 --local-dir data/tei/cache/bge-m3
```

### Git LFS

```bash
git lfs clone https://huggingface.co/BAAI/bge-m3 data/tei/cache/bge-m3
```

For an interrupted Git LFS clone, resume it in the existing directory:

```bash
git -C data/tei/cache/bge-m3 lfs pull
git -C data/tei/cache/bge-m3 lfs fsck
```

The Compose service mounts `./data/tei/cache` at `/data`, so both methods produce the model path required by `--model-id /data/bge-m3`.

## Start With Docker Compose

Copy the default port configuration, then start the service:

```bash
cp .env.example .env
docker compose up -d
```

`TEI_HTTP_PORT` in `.env` defaults to `8081`. The health check remains `starting` while the CPU model initializes; the first startup can take several minutes.

Follow the startup log when needed:

```bash
docker compose logs -f tei
```

## Verify

Wait until the health endpoint succeeds:

```bash
curl --fail --show-error http://127.0.0.1:8081/health
```

Then request an embedding through TEI's `/embed` endpoint:

```bash
curl --fail --show-error http://127.0.0.1:8081/embed \
  --header 'Content-Type: application/json' \
  --data '{"inputs":"Pomelo Orbit"}'
```

## Stop Or Update

```bash
docker compose down
docker compose pull
docker compose up -d
```

`down` removes the container but retains `data/tei/cache`, including the downloaded model.

## Orbit-Managed Deployment

For the `tei-bge-m3` Application created in Pomelo Orbit, use the persistent model directory below instead of this script's relative path:

```text
data/deployment/tei-bge-m3/default/tei/cache/bge-m3
```

Keep the Version component mount `tei/cache -> /data` and set its command arguments to `--model-id /data/bge-m3`. Deploy or restart the Service through Orbit only after the model download and integrity check finish. Orbit access is controlled by the Version expose and its Traefik route; the `8081` binding above applies only to this standalone Compose script.
