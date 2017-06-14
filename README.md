# Luzifer / vault-otp-ui

`vault-otp-ui` is a viewer for time based one-time passwords whose secret is stored in [Vault](https://vaultproject.io/). After the Github oAuth2 login the interface features a clean list of tokens with their corresponding account names, a (regular expression capable) filter function, automatic refresh of the shown tokens after they got invalid and a mobile-friendly interface which allows the usage on any mobile phone.

## Storage of the secrets

Two different methods are supported to store the secrets in Vault:

- Vault 0.7.x included [TOTP backend](https://www.vaultproject.io/docs/secrets/totp/index.html)
- Custom (generic) secrets containing `secret`, `name`, and `icon` keys
    - Icons supported are to be chosen from [FontAwesome](http://fontawesome.io/) icon set
    - When no `name` is set the Vault key will be used as a name

(When using the Vault builtin TOTP backend switching the icons for the tokens is not supported.)

## Setup

1. Create a new [oAuth application](https://github.com/settings/developers)
2. Configure `<your vault-otp-ui instance>/oauth2` as the callback URL
3. Configure the Github authentication backend for your users to be able to `read` the keys containing the secrets / TOTP codes
4. See `vault-otp-ui --help` for configuration parameters
    - You must configure the Github oAuth2 credentials
    - You must configure the Vault parameters
    - You should configure a `session-secret` having at least 64 byte length (If you don't set this it's chosen randomly which will invalidate your session cookies on every restart of the application)
