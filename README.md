#Build

```
git clone git@github.com:rawdigits/go-startapi.git
cd go-startapi
go build
```

#Usage

You'll need a file called `cert.p12` in the cwd. This is a special API client certificate from startcom. (different from the one you use in browser!)

Set the environment variables as follows:

```
export STARTCOM_API_CERT_PASSWORD=[password]
export STARTCOM_API_TOKEN_ID=[token_id]
```

`STARTCOM_API_CERT_PASSWORD` is the password for the cert.p12 file that identifies you to startcom.

`STARTCOM_API_TOKEN_ID` is the token identifier found [here](https://startssl.com/StartAPI/ApplyPart).

`./go-startapi -d [domain(s)]`

Optional:
  `-test` #uses apitest.startcom.com, which issues certs valid for 1 day. this is only for testing.

Go-startapi will generate a fresh RSA 4096 bit key, contact startcom, and write three files, the key, the certificate, and the intermediate certificate into the local directory.

#Examples

Generate a certificate for example.com:
```
./go-startapi -d example.com
```

Generate a wildcard certificate for dev.example.com:
```
./go-startapi -d "*.dev.example.com,dev.example.com"
```


#Notes

Startcom only allows you to issue three certs per CN in a 24 hour period, so don't test with important domain names.
