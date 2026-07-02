#!/usr/bin/env python3
"""生成 ULID"""

import argparse

import ulid

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="生成 ULID")
    parser.add_argument("-n", type=int, default=1, help="生成数量")
    args = parser.parse_args()

    for _ in range(args.n):
        print(ulid.new())
