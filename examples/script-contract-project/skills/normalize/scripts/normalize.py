from typing import Annotated, Literal, TypedDict

# Input "value" is linked from the imported Tool Output by Procedure with:.
class NormalizeOutput(TypedDict):
    normalized: Annotated[str, {"minLength": 1, "pattern": r"^[a-z0-9.-]+$"}]
    accepted: Literal[True]

def main(input: dict) -> NormalizeOutput:
    return {"normalized": input["value"].strip(), "accepted": True}
