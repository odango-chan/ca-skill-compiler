# /// connectionagent-tool
# name: normalize-dnn
# entrypoint: main
# inputSchema: NormalizeInput
# outputSchema: NormalizeOutput
# schemaMode: embedded
# effect: local_compute
# sandbox:
#   network: denied
# timeout: 1s
# interface:
#   input:
#     type: object
#     additionalProperties: false
#     required: [dnn]
#     properties:
#       dnn: {type: string, minLength: 1, pattern: "^[a-z0-9.-]+$"}
#   output:
#     type: object
#     additionalProperties: false
#     required: [dnn, accepted]
#     properties:
#       dnn: {type: string, minLength: 1}
#       accepted: {type: boolean}
# ///

def main(input: dict) -> dict:
    return {"dnn": input["dnn"], "accepted": True}
