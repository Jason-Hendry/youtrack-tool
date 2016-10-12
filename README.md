# youtrack-tool

## start.sh

create a runner
```
#!/bin/bash

GO_ROOT=/usr/local/go \
GO_PATH=~/rain/gantry-go/ \
YTTOOL_ID=<YOUR_APP_ID> \
YTTOOL_SECRET=<YOUR_APP_SECRET> \
YTTOOL_URL=https://<company>.myjetbrains.com/youtrack \
YTTOOL_PORT=8080 \
YTTOOL_REDIS=localhost:6380 \
YTTOOL_SCOPE=c844906d-ceff-4800-8ae6-00adfbde4a15 \
go run backend.go
```
