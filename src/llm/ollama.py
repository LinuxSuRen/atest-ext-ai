import logging
import requests
from typing import Dict, List, Any, Optional

from .base_connector import BaseConnector
from .prompt_builder import PromptBuilder

class OllamaConnector(BaseConnector):
    """Connector for Ollama LLM service"""
    
    def __init__(self, config):
        """Initialize Ollama connector
        
        Args:
            config: Configuration object or dictionary
        """
        super().__init__(config)
        self.llm_config = config.get_llm_config() if hasattr(config, 'get_llm_config') else config.get('llm', {})
        self.model = self.llm_config.get('model', 'llama3.2:1b')
        self.server_url = self.llm_config.get('server_url', 'http://localhost:11434')
        self.prompt_builder = PromptBuilder()
        
        logging.info(f"Initialized Ollama connector with model: {self.model}")
    
    def generate_content(self, prompt: str) -> str:
        """Generate SQL content using Ollama API
        
        Args:
            prompt: The prompt string for SQL generation
            
        Returns:
            Generated SQL query string
        """
        try:
            api_url = f"{self.server_url}/api/generate"
            
            payload = {
                "model": self.model,
                "prompt": prompt,
                "stream": False
            }
            
            response = requests.post(api_url, json=payload)
            response.raise_for_status()
            
            result = response.json()
            return result.get('response', '').strip()
        except Exception as e:
            logging.error(f"Error generating content from Ollama: {str(e)}")
            raise
    
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
        prompt = self.prompt_builder.build_sql_generation_prompt(
            natural_language_input=natural_language_input,
            database_type=database_type,
            schemas=schemas,
            examples=examples
        )
        
        return self.generate_content(prompt)