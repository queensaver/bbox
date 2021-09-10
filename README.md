# bbox
bbox is the server component to the bhive

Authenticate against the server by getting a token and saving it in a file called `~/.queensaver_token` (just the token, without the linefeed): 

```bash
curl --cookie-jar - -X POST -F 'username=your-username' -F 'password=your-password' https://api.queensaver.com/v1/login
```

Install the bbox software with the following command on your raspberry pi: 

```bash
curl https://raw.githubusercontent.com/queensaver/bbox/master/ansible/install.sh | bash
```
