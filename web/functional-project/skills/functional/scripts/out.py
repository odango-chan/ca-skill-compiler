from typing import NotRequired, TypedDict

Output = TypedDict(
    "Output",
    {
        "5qi": int,
        "optional label": NotRequired[str],
    },
)


def main(input: dict) -> Output:
    return {}
