[![Go Report Card](https://goreportcard.com/badge/github.com/Luzifer/vault-otp-ui)](https://goreportcard.com/report/github.com/Luzifer/vault-otp-ui)
![](https://badges.fyi/github/license/Luzifer/vault-otp-ui)
![](https://badges.fyi/github/downloads/Luzifer/vault-otp-ui)
![](https://badges.fyi/github/latest-release/Luzifer/vault-otp-ui)

# Luzifer / vault-otp-ui

`vault-otp-ui` is a viewer for time based one-time passwords whose secret is stored in [Vault](https://vaultproject.io/). After the Github oAuth2 login the interface features a clean list of tokens with their corresponding account names, a (regular expression capable) filter function, automatic refresh of the shown tokens after they got invalid and a mobile-friendly interface which allows the usage on any mobile phone. Additionally with all modern browsers you should be able to copy the one-time password into your clipboard with just one click!

## Storage of the secrets

Two different methods are supported to store the secrets in Vault:

- Vault 0.7.x included [TOTP backend](https://www.vaultproject.io/docs/secrets/totp/index.html)
- Custom (generic) secrets containing `secret`, `name`, `digits`, and `icon` keys
    - Icons supported are to be chosen from [FontAwesome](http://fontawesome.io/) icon set
    - When no `name` is set the Vault key will be used as a name
    - The `digits` field supports the values `6` (default) and `8` to generate longer 8-digit-codes

(When using the Vault builtin TOTP backend switching the icons for the tokens is not supported.)

## Setup

1. Create a new [oAuth application](https://github.com/settings/developers)
2. Configure `<your vault-otp-ui instance>/oauth2` as the callback URL
3. Configure the Github authentication backend for your users to be able to `read` the keys containing the secrets / TOTP codes
4. See `vault-otp-ui --help` for configuration parameters
    - You must configure the Github oAuth2 credentials
    - You must configure the Vault parameters
    - You should configure a `session-secret` having at least 64 byte length (If you don't set this it's chosen randomly which will invalidate your session cookies on every restart of the application)

## Security vs. Convenience

One of the key questions I found myself asking while developing this was whether to transmit the secrets used to generate the one-time passwords to the browser and to do the code generation in the browser or to keep the secrets in the backend application and only to deliver the codes themselves.

On the one hand the first solution would work when being offline because it can be cached in the browser. But seriously: I've never seen a OTP query when not being online so this wasn't a valid reason. On the other hand transmitting the secrets into the browser IMHO would be a major security flaw as - given the case you loose control over your browser having all those secrets stored in the local storage - an attacker would have the chance to generate unlimited one-time passwords for your accounts.

In the end I went with the solution to transmit only names and the currently valid code. This means being offline you are not able to generate a new code but also this means you can revoke access to the Vault keys and immediately stop the attackers ability to generate codes on your behalf.

----

![project status](https://d2o84fseuhwkxk.cloudfront.net/vault-otp-ui.svg)
