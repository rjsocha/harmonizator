# HARMONIZATOR

## Version 0.5.1

Allow to start some CLI task easly at the same time.

For example:

```
curl -sf hrm.wyga.it/hello:100 && echo "Hello World!"

# Run in other terminal or other host
curl -sf hrm.wyga.it/hello:100 && echo "Hello World!"

# ... repeat above command as many times you need - all commands will run at the same time
```

This will wait up to 100 seconds and trigger next command (echo in above example).

## Trigger mode

```
curl -sf hrm.wyga.it/hello && echo "Hello World!"
```

Trigger job via:

```
curl -sf hrm.wyga.it/hello:run
```
