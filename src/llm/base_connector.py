from abc import ABC, abstractmethod
from typing import Dict, List, Any, Optional

class BaseConnector(ABC):
    """Abstract base class for LLM connectors"""
    
    def __init__(self, config):
        """Initialize the connector with configuration
        
        Args:
            config: Configuration object or dictionary
        """
        self.config = config
    
    @abstractmethod
    def generate_content(self, prompt: str) -> str:
        """Generate content using the LLM
        
        Args:
            prompt: The prompt string for content generation
            
        Returns:
            Generated content string
        """
        pass
    
    @abstractmethod
    def generate_sql(
        self,
        natural_language_input: str,
        database_type: str,
        schemas: Optional[List[str]] = None,
        examples: Optional[List[Dict[str, str]]] = None
    ) -> str:
        """Generate SQL from natural language input
        
        Args:
            natural_language_input: User's natural language query
            database_type: Target database type
            schemas: Optional schema information
            examples: Optional example queries
            
        Returns:
            Generated SQL query
        """
        pass