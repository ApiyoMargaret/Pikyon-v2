import json
import os

def get_ts_type(schema):
    type_val = schema.get('type', 'string')
    if type_val == 'string':
        format_val = schema.get('format', '')
        if format_val == 'uuid':
            return 'string'
        return 'string'
    if type_val == 'integer':
        return 'number'
    if type_val == 'boolean':
        return 'boolean'
    if type_val == 'array':
        inner = schema.get('items', {}).get('type', 'string')
        if inner == 'string' and schema.get('items', {}).get('format') == 'uuid':
            return 'string[]'
        return f"{inner}[]"
    return 'any'

# Read swagger.json
with open('docs/swagger/swagger.json', 'r') as f:
    swagger = json.load(f)

# Get definitions (swagger 2.0 format)
definitions = swagger.get('components', {}).get('definitions', {})
if not definitions:
    definitions = swagger.get('definitions', {})

# Create output directory
os.makedirs('../frontend/src/api', exist_ok=True)

# Generate TypeScript interfaces
with open('../frontend/src/api/types.ts', 'w') as out:
    out.write("// Auto-generated TypeScript types from swagger.json\n")
    out.write("// Run `make sync-api` to regenerate\n\n")
    
    for name, schema in definitions.items():
        type_val = schema.get('type', 'string')
        properties = schema.get('properties', {})
        required = schema.get('required', [])
        
        if type_val == 'object' and properties:
            out.write(f"export interface {name} {{\n")
            for prop_name, prop_schema in properties.items():
                ts_type = get_ts_type(prop_schema)
                req_marker = " " if prop_name in required else "?"
                out.write(f"  {prop_name}{req_marker}: {ts_type}\n")
            out.write("}\n\n")
        else:
            out.write(f"// {name}: {type_val} - skipped (complex type)\n\n")

    print(f"Generated {len(definitions)} TypeScript type definitions")
