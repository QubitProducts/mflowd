[ -z "$MFLOWD_VERBOSE" ] || VERBOSE=-v
[ -z "$GOOGLE_APPLICATION_CREDENTIALS" ] && export GOOGLE_APPLICATION_CREDENTIALS=/etc/mflowd/key.json
/go/bin/mflowd $VERBOSE -p 6666 -s pubsub $MFLOWD_SUB
