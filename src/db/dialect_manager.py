import re
from typing import Dict, List, Optional
from enum import Enum

class DatabaseType(Enum):
    """Supported database types"""
    MYSQL = "mysql"
    POSTGRESQL = "postgresql"
    SQLITE = "sqlite"
    MSSQL = "mssql"
    ORACLE = "oracle"

class DialectManager:
    """Manager for handling database-specific SQL dialect adjustments"""
    
    def __init__(self):
        """Initialize the dialect manager"""
        self.dialect_rules = {
            DatabaseType.MYSQL: self._mysql_adjustments,
            DatabaseType.POSTGRESQL: self._postgresql_adjustments,
            DatabaseType.SQLITE: self._sqlite_adjustments,
            DatabaseType.MSSQL: self._mssql_adjustments,
            DatabaseType.ORACLE: self._oracle_adjustments
        }
    
    def adjust_sql_for_dialect(self, sql: str, database_type: str) -> str:
        """Adjust SQL query for specific database dialect
        
        Args:
            sql: The SQL query to adjust
            database_type: Target database type
            
        Returns:
            Adjusted SQL query
        """
        try:
            db_type = DatabaseType(database_type.lower())
            adjustment_func = self.dialect_rules.get(db_type)
            
            if adjustment_func:
                return adjustment_func(sql)
            else:
                return sql
        except ValueError:
            # Unknown database type, return original SQL
            return sql
    
    def _mysql_adjustments(self, sql: str) -> str:
        """Apply MySQL-specific adjustments
        
        Args:
            sql: Original SQL query
            
        Returns:
            MySQL-adjusted SQL query
        """
        # Replace double quotes with backticks for identifiers
        sql = re.sub(r'"([^"]+)"', r'`\1`', sql)
        
        # Replace LIMIT syntax if needed
        sql = re.sub(r'\bOFFSET\s+(\d+)\s+ROWS\s+FETCH\s+NEXT\s+(\d+)\s+ROWS\s+ONLY\b', 
                    r'LIMIT \2 OFFSET \1', sql, flags=re.IGNORECASE)
        
        # Replace boolean literals
        sql = re.sub(r'\bTRUE\b', '1', sql, flags=re.IGNORECASE)
        sql = re.sub(r'\bFALSE\b', '0', sql, flags=re.IGNORECASE)
        
        return sql
    
    def _postgresql_adjustments(self, sql: str) -> str:
        """Apply PostgreSQL-specific adjustments
        
        Args:
            sql: Original SQL query
            
        Returns:
            PostgreSQL-adjusted SQL query
        """
        # Replace backticks with double quotes for identifiers
        sql = re.sub(r'`([^`]+)`', r'"\1"', sql)
        
        # Replace LIMIT syntax
        sql = re.sub(r'\bLIMIT\s+(\d+)\s+OFFSET\s+(\d+)\b', 
                    r'OFFSET \2 ROWS FETCH NEXT \1 ROWS ONLY', sql, flags=re.IGNORECASE)
        
        # Ensure proper boolean literals
        sql = re.sub(r'\b1\b(?=\s*(=|!=|<>)\s*\w)', 'TRUE', sql)
        sql = re.sub(r'\b0\b(?=\s*(=|!=|<>)\s*\w)', 'FALSE', sql)
        
        return sql
    
    def _sqlite_adjustments(self, sql: str) -> str:
        """Apply SQLite-specific adjustments
        
        Args:
            sql: Original SQL query
            
        Returns:
            SQLite-adjusted SQL query
        """
        # SQLite is generally more permissive, minimal adjustments needed
        
        # Replace double quotes with square brackets for identifiers if needed
        # sql = re.sub(r'"([^"]+)"', r'[\1]', sql)
        
        # SQLite supports both TRUE/FALSE and 1/0 for booleans
        return sql
    
    def _mssql_adjustments(self, sql: str) -> str:
        """Apply SQL Server-specific adjustments
        
        Args:
            sql: Original SQL query
            
        Returns:
            SQL Server-adjusted SQL query
        """
        # Replace backticks with square brackets for identifiers
        sql = re.sub(r'`([^`]+)`', r'[\1]', sql)
        
        # Replace LIMIT with TOP clause
        limit_match = re.search(r'\bLIMIT\s+(\d+)\b', sql, flags=re.IGNORECASE)
        if limit_match:
            limit_value = limit_match.group(1)
            sql = re.sub(r'\bLIMIT\s+\d+\b', '', sql, flags=re.IGNORECASE)
            sql = re.sub(r'\bSELECT\b', f'SELECT TOP {limit_value}', sql, flags=re.IGNORECASE, count=1)
        
        return sql
    
    def _oracle_adjustments(self, sql: str) -> str:
        """Apply Oracle-specific adjustments
        
        Args:
            sql: Original SQL query
            
        Returns:
            Oracle-adjusted SQL query
        """
        # Replace backticks with double quotes for identifiers
        sql = re.sub(r'`([^`]+)`', r'"\1"', sql)
        
        # Replace LIMIT with ROWNUM
        limit_match = re.search(r'\bLIMIT\s+(\d+)\b', sql, flags=re.IGNORECASE)
        if limit_match:
            limit_value = limit_match.group(1)
            sql = re.sub(r'\bLIMIT\s+\d+\b', '', sql, flags=re.IGNORECASE)
            sql = f"SELECT * FROM ({sql}) WHERE ROWNUM <= {limit_value}"
        
        return sql
    
    def get_supported_databases(self) -> List[str]:
        """Get list of supported database types
        
        Returns:
            List of supported database type strings
        """
        return [db_type.value for db_type in DatabaseType]
    
    def validate_database_type(self, database_type: str) -> bool:
        """Validate if database type is supported
        
        Args:
            database_type: Database type string to validate
            
        Returns:
            True if supported, False otherwise
        """
        try:
            DatabaseType(database_type.lower())
            return True
        except ValueError:
            return False