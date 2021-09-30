# Cookie Setter

Microservice for getting cookie values based off a config file based off canary weighting

## Usage

Add an app to `cookie-setter.yaml`. Specify the `percent` of traffic you want to receive the `ifSuccessful` cookie vs. the `ifFail` cookie.

For example,

```yaml
- appName: jupyterhub
  expiration: 48h
  percent: .90
  ifSuccessful: # less than percent
    key: a
    value: a
  ifFail:
    key: b
    value: b
```

The above configuration states that for 90% of `PUT`s to `/` with `{"app": "jupyterhub"}` will return `{"Key": "a", "Value": "a", Expiration: <48 hours from now>}`. For the 10% case, the server will return `{"Key": "b", "Value": "b", Expiration: <48 hours from now>}`.

*Note* config is hot loaded every time `/` is called, so modify the config will load all new changes upon the next request.