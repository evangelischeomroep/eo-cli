# eo-cli

A command-line tool for EO (Evangelische Omroep) developers to automate common tasks.

## Requirements

- [Go](https://golang.org/) 1.25+
- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) (logged in via `az login`)

## Installation

```bash
go build -o eo ./cmd/eo
```

## Commands

### `pim`

Requests the Contributor role on the EO Studio Digitaal Azure subscription for 8 hours via Azure PIM (Privileged Identity Management).

```bash
eo pim [justification]
```

**Arguments:**

| Argument        | Required | Default                                         |
| --------------- | -------- | ----------------------------------------------- |
| `justification` | No       | `Requesting access to perform necessary tasks.` |

**Example:**

```bash
eo pim "Deploying hotfix to production"
```
