# scripts

This directory contains scripts that are used to convert data from the bespoken formats of the original DikuMUD to the format used by this MUD.

## Requirements

- Python 3.10+
- `requirements.txt`
    ```bash
    pip install -r requirements.txt
    ```

## `area_converter.py`
Converts a DikuMUD area file to a YAML format. This is a quick, hacky way of converting a DikuMUD area file to a YAML format. This particular script is only designed to extract the rooms from the area file, and nothing else. It does not handle the other sections of the area file, such as the objects or mobs.

### Usage

```bash
python area_converter.py <filename> [area_name]
```

