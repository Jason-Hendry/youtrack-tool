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
go run main
```
