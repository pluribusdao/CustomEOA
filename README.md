# CustomEOA by Pluribus DAO

CustomEOA is a lightweight vanity EOA generator for Ethereum.
It searches for an address that matches a pattern you define.

The pattern supports `?` as a wildcard character.

## Contributing

Contributions are welcome.
If you want to improve the tool, open an issue or submit a pull request.

## Local Usage

Install dependencies and create your local environment file:

```bash
go mod download
cp .env.example .env
```

Configure `.env`:

```env
ADDRESS_PATTERN=dead????????????????????????????????beef
WORKERS=0
```

- `ADDRESS_PATTERN`: optional `0x`, must be exactly 40 hex characters.
- Supported characters: lowercase hex (`a-f`) and `?`.
- `WORKERS=0`: uses the available CPU core count.

Run:

```bash
go run .
```

Sample output:

```text
address=0x...
public_key=...
private_key=...
```

## License

This project currently does not include a license file.
