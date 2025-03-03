# mob_resets_converter.py
#
# Converting mob resets from a DikuMUD area file into a YAML format.
#
# Usage: python mob_resets_converter.py <filename>
import re
import sys

def parse_resets(input_text):
    """Parse DikuMUD resets section to extract mobile resets"""
    
    # Check if we have a #RESETS section
    if "#RESETS" not in input_text:
        print("No #RESETS section found in the input text")
        return None
    
    # Extract the #RESETS section
    resets_pattern = r'#RESETS(.*?)(?=#|\Z)'
    resets_match = re.search(resets_pattern, input_text, re.DOTALL)
    
    if not resets_match:
        print("Could not extract #RESETS section")
        return None
    
    resets_section = resets_match.group(1).strip()
    
    # This will hold all our parsed mobile resets
    mob_resets = []
    
    # Process each line in the resets section
    for line in resets_section.split('\n'):
        line = line.strip()
        
        # Skip empty lines
        if not line:
            continue
        
        # Look for lines that start with 'M'
        if line.startswith('M '):
            # Extract the mobile reset data
            # Proper format: M 0 <mob-vnum> <limit> <room-vnum> <max_world> * comment
            parts = line.split('*', 1)
            reset_data = parts[0].strip().split()
            
            # Get the comment if it exists
            comment = parts[1].strip() if len(parts) > 1 else ""
            
            # Ensure we have enough elements (M 0 vnum limit room_vnum)
            if len(reset_data) >= 5:
                mob_reset = {
                    'mob_vnum': reset_data[2],
                    'limit': reset_data[3],  # Local limit
                    'room_vnum': reset_data[4],
                    'max_world': reset_data[5] if len(reset_data) > 5 else "1",  # Global limit
                    'comment': comment
                }
                mob_resets.append(mob_reset)
    
    return mob_resets

def generate_yaml(mob_resets):
    """Generate YAML output from parsed mobile resets"""
    yaml_lines = []
    yaml_lines.append("mob_resets:")
    
    for reset in mob_resets:
        yaml_lines.append(f"  - mob_vnum: {reset['mob_vnum']}")
        yaml_lines.append(f"    room_vnum: {reset['room_vnum']}")
        yaml_lines.append(f"    limit: {reset['limit']}")
        if 'max_world' in reset:
            yaml_lines.append(f"    max_world: {reset['max_world']}")
        if reset['comment']:
            yaml_lines.append(f"    comment: \"{reset['comment']}\"")
    
    return '\n'.join(yaml_lines)

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
        
        # Parse mob resets
        mob_resets = parse_resets(input_text)
        if mob_resets:
            # Generate YAML for mob resets
            resets_yaml = generate_yaml(mob_resets)
            
            # Write to output file
            output_file = file_path.rsplit('.', 1)[0] + '-resets.yml'
            with open(output_file, 'w', encoding='utf-8') as f:
                f.write(f"---\nname: Midgaard\n{resets_yaml}")
            
            print(f"Created YAML with mob resets in {output_file}")
            
            # Also print the result to console for verification
            print("\nExtracted mob resets:")
            print(resets_yaml)
        else:
            print("No mob resets were found to convert")
    
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()