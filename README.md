# Akula

Akula provides a convenient cli to search for and retrieve stealer log data from the Akula Telegram bot. It handles authentication, message sending, and response parsing, allowing you to quickly access the data you need.

## Installation

```bash
# Install directly using Go
go install github.com/yourusername/akula@latest

# Or clone and build manually
git clone https://github.com/yourusername/akula.git
cd akula
go build
```

## Authentication

*If you don't want to use your account for this message [Gnome](https://t.me/gnome_gl) and he will give you a key to use.*

To use this application with Telegram, you'll need to obtain API credentials. Here's how:

1. Visit [my.telegram.org](https://my.telegram.org/auth) and log in with your phone number
2. Click on 'API development tools'
3. Fill in the form with any name and description (the website URL is optional)
4. You will receive:
   - `api_id` (a number)
   - `api_hash` (a string)

### Required Credentials
You'll need these three pieces of information:
- API ID
- API Hash
- Phone Number (in international format, e.g., `+1234567890`)

### Setting Up Credentials

On your first run of the cli it will walk you through a wizard to create the config file for you

It will generate a config File in `~/.config/akula/config.json`
```json
{
    "tg_api_id": <ID>,
    "tg_api_hash": <HASH>,
    "phone_number": <PHONE>
}
```

It will also create a `session.json` file in the config folder which is what it will use for subsequent connections. 

## Usage

```
akula [flags] [search term]
```

### Flags

- `--api-id`: Telegram API ID
- `--api-hash`: Telegram API Hash
- `--phone`: Phone number for Telegram login
- `--wait`: Time to wait for response in seconds (default: 30)
- `-v, --verbose`: Enable verbose output

### Examples

Search for a specific term:
```
akula "search term"
```

Search with custom wait time:
```
akula --wait 60 "search term"
```
