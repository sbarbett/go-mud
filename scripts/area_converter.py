# area_converter.py
#
# This script is a quick, hacky way of converting a DikuMUD area file
# into a YAML format. This particular script is only designed to extract
# the rooms from the area file, and nothing else. It does not handle    
# the other sections of the area file, such as the objects or mobs.
#
# Usage: python area_converter.py <filename> [area_name]
#
# If area_name is not provided, it will default to "Midgaard".
import re
import sys

def parse_rooms(input_text):
    """Parse DikuMUD room definitions into a structured format"""
    
    # First check if we have a #ROOMS section
    if "#ROOMS" not in input_text:
        print("No #ROOMS section found in the input text")
        return None
        
    # Get only the ROOMS section
    rooms_section = input_text.split("#ROOMS")[1].strip()
    
    # Find the end of the ROOMS section if there are other sections
    if "#" in rooms_section:
        sections = re.findall(r'\n#[A-Z]+', rooms_section)
        if sections:
            for section in sections:
                if section != "\n#ROOMS":
                    end_pos = rooms_section.find(section)
                    if end_pos > 0:
                        rooms_section = rooms_section[:end_pos]
                        break
    
    # This will hold all our parsed rooms
    rooms = {}
    
    # Split into individual room blocks
    room_blocks = re.split(r'\n#(\d+)', rooms_section)
    room_blocks = room_blocks[1:]  # Skip the first empty element
    
    # Process room blocks in pairs (room_id, room_content)
    for i in range(0, len(room_blocks), 2):
        if i+1 >= len(room_blocks):
            break
            
        room_id = room_blocks[i]
        room_content = room_blocks[i+1].strip()
        
        # Parse the room content
        room_data = parse_room_content(room_id, room_content)
        if room_data:
            rooms[room_id] = room_data
    
    return rooms

def parse_room_content(room_id, content):
    """Parse a single room's content"""
    lines = content.split('\n')
    
    # Room name is the first line (without the trailing ~)
    name = lines[0].rstrip('~')
    
    # Find where the description ends (at a standalone ~)
    desc_end = -1
    for i, line in enumerate(lines[1:], 1):
        if line == '~':
            desc_end = i
            break
    
    if desc_end == -1:
        print(f"Warning: Could not find end of description for room {room_id}")
        return None
    
    # Extract the description
    description = '\n'.join(lines[1:desc_end])
    
    # Current position in the content
    pos = desc_end + 1
    
    # Skip room flags/sector type
    if pos < len(lines) and not (lines[pos].startswith('D') or 
                                lines[pos].startswith('E') or 
                                lines[pos].startswith('S')):
        pos += 1
    
    # Parse exits and extra descriptions
    exits = {}
    environment = []
    
    # Direction mapping
    dir_map = {
        'D0': 'north',
        'D1': 'east', 
        'D2': 'south',
        'D3': 'west',
        'D4': 'up',
        'D5': 'down'
    }
    
    while pos < len(lines):
        line = lines[pos]
        
        # End of room
        if line.startswith('S'):
            break
        
        # Exit direction
        elif line.startswith('D'):
            dir_code = line[:2]
            direction = dir_map.get(dir_code, 'unknown')
            pos += 1
            
            # Exit description
            exit_desc_end = -1
            for i, l in enumerate(lines[pos:], pos):
                if l == '~':
                    exit_desc_end = i
                    break
            
            if exit_desc_end == -1:
                print(f"Warning: Could not find end of exit description in room {room_id}")
                break
                
            exit_desc = '\n'.join(lines[pos:exit_desc_end])
            pos = exit_desc_end + 1
            
            # Skip keywords (Door/gate name)
            keyword_end = -1
            for i, l in enumerate(lines[pos:], pos):
                if l == '~':
                    keyword_end = i
                    break
            
            if keyword_end == -1:
                print(f"Warning: Could not find end of exit keywords in room {room_id}")
                break
                
            pos = keyword_end + 1
            
            # Get door flags and destination
            if pos < len(lines):
                door_info = lines[pos].strip().split()
                pos += 1
                
                if len(door_info) >= 3:
                    destination = door_info[2]
                    # Only add if not a closed exit
                    if destination != "-1":
                        exits[direction] = {
                            'id': destination,
                            'description': exit_desc
                        }
        
        # Extra descriptions
        elif line.startswith('E'):
            # Get the keywords
            keyword_line = line[1:].strip()
            pos += 1
            
            # If keywords were not on the E line, they're on the next line
            if not keyword_line or '~' not in keyword_line:
                keyword_line = lines[pos]
                pos += 1
            
            keywords = keyword_line.split('~')[0].strip().split()
            
            # Find the end of the extra description
            extra_desc_end = -1
            for i, l in enumerate(lines[pos:], pos):
                if l == '~':
                    extra_desc_end = i
                    break
            
            if extra_desc_end == -1:
                print(f"Warning: Could not find end of extra description in room {room_id}")
                break
                
            extra_desc = '\n'.join(lines[pos:extra_desc_end])
            pos = extra_desc_end + 1
            
            environment.append({
                'keywords': keywords,
                'description': extra_desc
            })
        
        else:
            # Unknown line, skip
            pos += 1
    
    # Create the room data structure
    room_data = {
        'name': name,
        'description': description
    }
    
    if exits:
        room_data['exits'] = exits
        
    if environment:
        room_data['environment'] = environment
        
    return room_data

def generate_yaml(rooms, area_name="Midgaard"):
    """Generate YAML output from parsed rooms"""
    yaml_lines = []
    yaml_lines.append(f"name: {area_name}")
    yaml_lines.append("rooms:")
    
    # Sort rooms by room ID numerically
    sorted_room_ids = sorted(rooms.keys(), key=lambda x: int(x))
    
    for room_id in sorted_room_ids:
        room = rooms[room_id]
        yaml_lines.append(f"  {room_id}:")
        yaml_lines.append(f'    name: "{room["name"]}"')
        
        # Format description with pipe syntax for multiline
        yaml_lines.append("    description: |")
        for line in room["description"].split('\n'):
            yaml_lines.append(f"      {line}")
        
        # Format exits
        if "exits" in room and room["exits"]:
            yaml_lines.append("    exits:")
            for direction, exit_data in room["exits"].items():
                yaml_lines.append(f"      {direction}:")
                yaml_lines.append(f"        id: {exit_data['id']}")
                # Format exit description
                exit_desc = exit_data["description"].replace('\n', ' ').strip()
                yaml_lines.append(f'        description: "{exit_desc}"')
        
        # Format environment/extra descriptions
        if "environment" in room and room["environment"]:
            yaml_lines.append("    environment:")
            for env in room["environment"]:
                # Format keywords as an array
                keywords_str = ', '.join([f'"{k}"' for k in env["keywords"]])
                yaml_lines.append(f"      - keywords: [{keywords_str}]")
                
                # Format description with pipe syntax
                yaml_lines.append("        description: |")
                for line in env["description"].split('\n'):
                    yaml_lines.append(f"          {line}")
    
    return '\n'.join(yaml_lines)

def diku_to_yaml(input_text, area_name="Midgaard"):
    """Convert DikuMUD area file to YAML format"""
    # Parse the rooms
    rooms = parse_rooms(input_text)
    if not rooms:
        return "Error: Could not parse rooms from input text"
    
    # Generate YAML
    return generate_yaml(rooms, area_name)

def main():
    # Check if a file path was provided
    if len(sys.argv) < 2:
        print("Usage: python script.py <filename> [area_name]")
        sys.exit(1)
    
    # Get the file path and optional area name
    file_path = sys.argv[1]
    area_name = sys.argv[2] if len(sys.argv) > 2 else "Midgaard"
    
    try:
        # Read the input file
        with open(file_path, 'r', encoding='utf-8') as f:
            input_text = f.read()
        
        # Convert to YAML
        yaml_output = diku_to_yaml(input_text, area_name)
        
        # Write to output file
        output_file = file_path.rsplit('.', 1)[0] + '.yml'
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write('---\n' + yaml_output)
        
        print(f"Conversion complete! Output written to {output_file}")
    
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()