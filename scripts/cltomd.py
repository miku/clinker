import json
import sys
from datetime import datetime
import fileinput

def json_lines_to_markdown(json_lines):
    headers = ["Link", "Comment", "Elapsed (s)", "Timestamp"]
    markdown_table = "| " + " | ".join(headers) + " |\n"
    markdown_table += "| " + " | ".join(["---"] * len(headers)) + " |\n"

    for line in json_lines:
        try:
            # Parse the JSON line
            data = json.loads(line)
            timestamp = data["t"]
            # Parse ISO 8601 formatted timestamp without timezone
            timestamp_parsed = datetime.strptime(timestamp[:26], "%Y-%m-%dT%H:%M:%S.%f")
            elapsed_seconds = data["elapsed"] / 1e9  # Convert nanoseconds to seconds
            comment = data["comment"]
            link = data["link"]
            if "no such host" not in comment:
                continue

            # Create a row in the markdown table
            row = f"| {link} | {comment} | {elapsed_seconds:.2f} | {timestamp_parsed.isoformat()} |"
            markdown_table += row + "\n"
        except Exception as e:
            print(f"Failed to parse JSON line: {e}")
            continue
    return markdown_table


lines = [line for line in fileinput.input()]
markdown_table = json_lines_to_markdown(lines)
print(markdown_table)

