import re
import logging
from typing import List, Optional

def clean_sql_query(sql: str) -> str:
    """Clean and format SQL query
    
    Args:
        sql: Raw SQL query string
        
    Returns:
        Cleaned SQL query
    """
    if not sql:
        return ""
    
    # Remove markdown code blocks if present
    sql = re.sub(r'^```sql\s*', '', sql, flags=re.IGNORECASE | re.MULTILINE)
    sql = re.sub(r'^```\s*', '', sql, flags=re.MULTILINE)
    sql = re.sub(r'```$', '', sql, flags=re.MULTILINE)
    
    # Remove extra whitespace and normalize
    sql = re.sub(r'\s+', ' ', sql.strip())
    
    # Ensure query ends with semicolon if it doesn't already
    if sql and not sql.rstrip().endswith(';'):
        sql = sql.rstrip() + ';'
    
    return sql

def validate_sql_syntax(sql: str) -> bool:
    """Basic SQL syntax validation
    
    Args:
        sql: SQL query to validate
        
    Returns:
        True if basic syntax appears valid, False otherwise
    """
    if not sql or not sql.strip():
        return False
    
    # Clean the SQL first
    cleaned_sql = clean_sql_query(sql).upper().strip()
    
    # Check for basic SQL keywords
    sql_keywords = ['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER']
    starts_with_keyword = any(cleaned_sql.startswith(keyword) for keyword in sql_keywords)
    
    if not starts_with_keyword:
        return False
    
    # Basic parentheses matching
    open_parens = cleaned_sql.count('(')
    close_parens = cleaned_sql.count(')')
    
    return open_parens == close_parens

def extract_table_names(sql: str) -> List[str]:
    """Extract table names from SQL query
    
    Args:
        sql: SQL query string
        
    Returns:
        List of table names found in the query
    """
    table_names = []
    
    # Simple regex patterns for common table references
    patterns = [
        r'FROM\s+([\w`"\[\]]+)',  # FROM table
        r'JOIN\s+([\w`"\[\]]+)',  # JOIN table
        r'UPDATE\s+([\w`"\[\]]+)',  # UPDATE table
        r'INTO\s+([\w`"\[\]]+)',  # INSERT INTO table
    ]
    
    sql_upper = sql.upper()
    
    for pattern in patterns:
        matches = re.findall(pattern, sql_upper)
        for match in matches:
            # Clean table name (remove quotes, backticks, brackets)
            clean_name = re.sub(r'[`"\[\]]', '', match)
            if clean_name and clean_name not in table_names:
                table_names.append(clean_name)
    
    return table_names

def format_error_response(error_message: str, error_code: str = "GENERATION_ERROR") -> dict:
    """Format error response for gRPC
    
    Args:
        error_message: Human-readable error message
        error_code: Error code identifier
        
    Returns:
        Formatted error response dictionary
    """
    return {
        "error": {
            "code": error_code,
            "message": error_message
        }
    }

def log_sql_generation(natural_language: str, generated_sql: str, database_type: str):
    """Log SQL generation for debugging and monitoring
    
    Args:
        natural_language: Original natural language input
        generated_sql: Generated SQL query
        database_type: Target database type
    """
    logging.info(
        f"SQL Generation - DB: {database_type}, "
        f"Input: '{natural_language[:100]}...', "
        f"Output: '{generated_sql[:100]}...'"
    )