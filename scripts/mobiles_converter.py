# mobiles_converter.py
#
# Converting mobiles from a DikuMUD area file into a YAML format.
#
# Usage: python mobiles_converter.py <filename>
import re
import sys

def parse_mobiles(input_text):
    """Parse DikuMUD mobile definitions into a structured format"""
    
    # Check if we have a #MOBILES section
    if "#MOBILES" not in input_text:
        print("No #MOBILES section found in the input text")
        return None
    
    # Extract the #MOBILES section
    mobiles_pattern = r'#MOBILES(.*?)(?=#OBJECTS|\Z)'
    mobiles_match = re.search(mobiles_pattern, input_text, re.DOTALL)
    
    if not mobiles_match:
        print("Could not extract #MOBILES section")
        return None
    
    mobiles_section = mobiles_match.group(1).strip()
    
    # This will hold all our parsed mobiles
    mobiles = {}
    
    # Split into individual mobile blocks
    mobile_blocks = re.split(r'\n#(\d+)', mobiles_section)
    mobile_blocks = mobile_blocks[1:]  # Skip the first empty element
    
    # Process mobile blocks in pairs (mobile_id, mobile_content)
    for i in range(0, len(mobile_blocks), 2):
        if i+1 >= len(mobile_blocks):
            break
            
        mobile_id = mobile_blocks[i]
        mobile_content = mobile_blocks[i+1].strip()
        
        # Skip the ending #0 if present
        if mobile_id == '0':
            continue
        
        # Parse the mobile content
        mobile_data = parse_mobile_content(mobile_id, mobile_content)
        if mobile_data:
            mobiles[mobile_id] = mobile_data
    
    return mobiles

def parse_mobile_content(mobile_id, content):
    """Parse a single mobile's content"""
    lines = content.split('\n')
    
    if not lines:
        return None
    
    # Keywords are the first line (without the trailing ~)
    keywords = lines[0].rstrip('~').split()
    
    # Get short description (second line)
    short_desc = lines[1].rstrip('~') if len(lines) > 1 else ""
    
    # Find where the long description ends (at a standalone ~)
    long_desc_lines = []
    pos = 2
    while pos < len(lines) and lines[pos] != '~':
        long_desc_lines.append(lines[pos])
        pos += 1
    
    long_desc = '\n'.join(long_desc_lines)
    pos += 1  # Skip the ~
    
    # Find where the full description ends (at a standalone ~)
    full_desc_lines = []
    while pos < len(lines) and lines[pos] != '~':
        full_desc_lines.append(lines[pos])
        pos += 1
    
    full_desc = '\n'.join(full_desc_lines)
    pos += 1  # Skip the ~
    
    # Extract race
    race = lines[pos].rstrip('~') if pos < len(lines) else ""
    pos += 1
    
    # Extract level from the stats line (sixth field)
    # Format: act affected alignment type
    _ = lines[pos] if pos < len(lines) else ""
    pos += 1
    
    # Get level from first field of stats line
    stats_line = lines[pos] if pos < len(lines) else ""
    stats_parts = stats_line.split()
    level = stats_parts[0] if len(stats_parts) > 0 else "0"
    
    # Create the simplified mobile data structure
    mobile_data = {
        'keywords': keywords,
        'short_description': short_desc,
        'long_description': long_desc,
        'description': full_desc,
        'race': race,
        'level': level
    }
    
    return mobile_data

def generate_yaml(mobiles):
    """Generate YAML output from parsed mobiles"""
    yaml_lines = []
    yaml_lines.append("mobiles:")
    
    # Sort mobiles by ID numerically
    sorted_mobile_ids = sorted(mobiles.keys(), key=lambda x: int(x))
    
    for mobile_id in sorted_mobile_ids:
        mobile = mobiles[mobile_id]
        yaml_lines.append(f"  {mobile_id}:")
        
        # Keywords as an array
        keywords_str = ', '.join([f'"{k}"' for k in mobile['keywords']])
        yaml_lines.append(f"    keywords: [{keywords_str}]")
        
        # Short description
        yaml_lines.append(f'    short_description: "{mobile["short_description"]}"')
        
        # Long description with pipe syntax
        yaml_lines.append("    long_description: |")
        for line in mobile["long_description"].split('\n'):
            yaml_lines.append(f"      {line}")
        
        # Full description with pipe syntax
        yaml_lines.append("    description: |")
        for line in mobile["description"].split('\n'):
            yaml_lines.append(f"      {line}")
        
        # Race
        yaml_lines.append(f'    race: "{mobile["race"]}"')
        
        # Level
        yaml_lines.append(f"    level: {mobile['level']}")
    
    return '\n'.join(yaml_lines)

def merge_yaml_sections(rooms_yaml, mobiles_yaml):
    """Merge rooms and mobiles YAML sections"""
    # If rooms_yaml already has a '---' prefix, remove it for merging
    if rooms_yaml.startswith('---\n'):
        rooms_yaml = rooms_yaml[4:]
    
    # Extract the area name from rooms YAML
    area_name_match = re.search(r'name: (.+)', rooms_yaml)
    area_name = area_name_match.group(1) if area_name_match else "Midgaard"
    
    # Create the combined YAML
    combined_yaml = f"---\nname: {area_name}\n"
    
    # Add rooms section
    rooms_content = re.search(r'rooms:(.*?)(?=\Z|\nmobiles:)', rooms_yaml, re.DOTALL)
    if rooms_content:
        combined_yaml += f"rooms:{rooms_content.group(1)}"
    else:
        # If no rooms found, just add the mobiles
        combined_yaml += mobiles_yaml
        return combined_yaml
    
    # Add mobiles section
    mobiles_content = re.search(r'mobiles:(.*)', mobiles_yaml, re.DOTALL)
    if mobiles_content:
        combined_yaml += f"\nmobiles:{mobiles_content.group(1)}"
    
    return combined_yaml

def main():
    # Check if a file path was provided
    if len(sys.argv) < 2:
        print("Usage: python script.py <filename>")
        sys.exit(1)
    
    # Get the file path
    file_path = sys.argv[1]
    
    try:
        # Read the input file
        with open(file_path, 'r', encoding='utf-8') as f:
            input_text = f.read()
        
        # Parse mobiles
        mobiles = parse_mobiles(input_text)
        if mobiles:
            # Generate YAML for mobiles
            mobiles_yaml = generate_yaml(mobiles)
            
            # Read existing rooms YAML if it exists
            output_file = file_path.rsplit('.', 1)[0] + '.yml'
            try:
                with open(output_file, 'r', encoding='utf-8') as f:
                    rooms_yaml = f.read()
                
                # Merge rooms and mobiles
                combined_yaml = merge_yaml_sections(rooms_yaml, mobiles_yaml)
                
                # Write combined YAML
                with open(output_file, 'w', encoding='utf-8') as f:
                    f.write(combined_yaml)
                
                print(f"Added mobiles to existing YAML in {output_file}")
            
            except FileNotFoundError:
                # No existing rooms YAML, just write mobiles
                with open(output_file, 'w', encoding='utf-8') as f:
                    f.write(f"---\nname: Midgaard\n{mobiles_yaml}")
                
                print(f"Created new YAML with mobiles in {output_file}")
        else:
            print("No mobiles were found to convert")
    
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()