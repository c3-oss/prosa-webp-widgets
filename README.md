# prosa-webp-widgets

Static WebP widgets for GitHub profile READMEs, powered by
[`prosa`](https://github.com/c3-oss/prosa) analytics.

The renderer produces five same-size widgets:

- `overview.webp`
- `agent-mix.webp`
- `model-spend.webp`
- `project-focus.webp`
- `delegation.webp`

Each image is rendered at `2260x696` pixels from a `1130x348` logical widget.

## Preview With Mock Data

```bash
just run render --mock --out ./out/mock
```

The command writes all five files under `out/mock/`.

## Render From Prosa

```bash
export PROSA_SERVER_URL='https://prosa.example.com'
export PROSA_APP_TOKEN='prosa_app_...'

just run render --last 30d --out ./out/prod
```

## Upload To S3

```bash
export S3_BUCKET_NAME='your-bucket'
export AWS_REGION='us-east-1'
export AWS_ACCESS_KEY_ID='...'
export AWS_SECRET_ACCESS_KEY='...'
export S3_PREFIX='prosa-widgets'
export S3_PUBLIC_BASE_URL='https://cdn.example.com'

just run render --last 30d --out ./out/prod --upload
```

Optional:

- `PROSA_HTTP_TIMEOUT` or `--timeout` to control remote fetch timeout (default `30s`).
- `S3_ENDPOINT_URL` for S3-compatible storage.
- `S3_ACL` for buckets that still require a canned ACL.
- `CHROMIUM_BROWSER_BINARY_PATH` to use an existing Chrome/Chromium binary.

## Development

```bash
devbox shell
just test
just build
```

## License

[CC0 1.0 Universal](LICENSE).
