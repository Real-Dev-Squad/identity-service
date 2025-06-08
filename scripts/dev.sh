case "$OSTYPE" in
  msys*|cygwin*|win32*)
    SAM_CMD="sam.cmd"
    ;;
  *)
    SAM_CMD="sam"
    ;;
esac

go mod tidy
"$SAM_CMD" build
"$SAM_CMD" local start-api --env-vars env.json