# bbox
bbox is the server component to the bhive

Authenticate against the server by getting a token and saving it in a file called `~/.queensaver_token` (without the linefeed): 

```bash
curl --cookie-jar - -X POST -F 'username=queensaver@queensaver.com' -F 'password=bmajagmbh' https://api.queensaver.com/v1/login
```
