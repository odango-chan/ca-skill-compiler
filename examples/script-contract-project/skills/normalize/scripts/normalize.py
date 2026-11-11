from typing import Annotated, Literal, NotRequired, TypedDict

# This is a compile-only test fixture. No custom runtime library is required.
class DnnScope(TypedDict):
    sst: Annotated[int, {"minimum": 0, "maximum": 255}]
    sd: NotRequired[Annotated[int, {"minimum": 0, "maximum": 16777214}]]

class NormalizeInput(TypedDict):
    dnn: Annotated[str, {"minLength": 1, "pattern": r"^[a-z0-9][a-z0-9.-]*$"}]
    sNssai: DnnScope
    requestedPduSessionType: Literal["IPV4"]

class NormalizeOutput(TypedDict):
    dnn: Annotated[str, {"minLength": 1}]
    sNssai: DnnScope
    accepted: Literal[True]

def main(input: NormalizeInput) -> NormalizeOutput:
    return {
        "dnn": input["dnn"],
        "sNssai": input["sNssai"],
        "accepted": True,
    }
