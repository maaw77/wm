#!/usr/bin/env python3
import json
import sys
import urllib.error
import urllib.request

BASE_URL = "http://127.0.0.1:8080"


def post_json(url: str, payload: dict, timeout_s: float = 10.0) -> tuple[int, dict]:
    body = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout_s) as resp:
            resp_body = resp.read().decode("utf-8")
            return resp.getcode(), json.loads(resp_body)
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {e.code} from {url}: {raw}") from e
    except urllib.error.URLError as e:
        raise RuntimeError(f"Request failed: {e}") from e


def assert_eq(got, want, msg: str):
    if got != want:
        raise AssertionError(f"{msg}: got={got!r}, want={want!r}")


def main() -> int:
    links = [
        "google.com",
        "yandex.ru",
        "nonexistent.invalid",
        "bad host name !!!",
    ]

    expected = {
        "google.com": "available",
        "yandex.ru": "available",
        "nonexistent.invalid": "not available",
        "bad host name !!!": "not available",
    }

    status, resp = post_json(f"{BASE_URL}/links", {"links": links})
    assert_eq(status, 200, "POST /links status code")

    if "links_num" not in resp or "links" not in resp:
        raise AssertionError(f"Bad response schema: {resp}")

    task_id = resp["links_num"]
    if not isinstance(task_id, int) or task_id <= 0:
        raise AssertionError(f"links_num must be positive int, got: {task_id!r}")

    links_map = resp["links"]
    if not isinstance(links_map, dict):
        raise AssertionError(f"links must be object/dict, got: {type(links_map)}")

    for u in links:
        if u not in links_map:
            raise AssertionError(f"Missing link in response: {u}")
        got_status = links_map[u]
        want_status = expected[u]
        assert_eq(got_status, want_status, f"Status mismatch for {u}")

    print("OK")
    print(f"task_id={task_id}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
