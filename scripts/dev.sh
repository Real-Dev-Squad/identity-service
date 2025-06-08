if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" || "$OSTYPE" == "win32" ]]; then
    SAM_CMD="sam.cmd"
else
    SAM_CMD="sam"
fi

go mod tidy
$SAM_CMD build
$SAM_CMD local start-api --env-vars env.json