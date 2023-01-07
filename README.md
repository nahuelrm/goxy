# Goxy

`Goxy` is a tool written in go that makes use of https://whoxy.com free features for finding root domains that belong to the same company. 

It's very useful to expand your scope.

## Installation

```
go install github.com/nahuelrm/goxy@latest
```

## Dependencies

- [httprobe](https://github.com/tomnomnom/httprobe)
- [htmlq](https://github.com/mgdm/htmlq)
- [anew](https://github.com/tomnomnom/anew)

## Usage

| Options | Description |
| :--- | :--- |
| `-d <domain>` | Start scan domain |
| `-c <int>` | Set the concurrency level. (default 20) |
| `--complete` | Complete scan mode |
| `--keyword <word>` | Keywords only scan mode |

