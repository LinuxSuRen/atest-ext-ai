import logging
import openai
from typing import Dict, Any, Optional, List

from .base_connector import BaseConnector
from .prompt_builder import PromptBuilder

class OpenAIConnector(BaseConnector):
    """Connector for OpenAI API"""
    
    def __init__(self, config):
        """Initialize OpenAI connector
        
        Args:
            config: Configuration object or dictionary
        """
        super().__init__(config)
        self.llm_config = config.get_llm_config() if hasattr(config, 'get_llm_config') else config.get('llm', {})
        self.model = self.llm_config.get('model', 'gpt-3.5-turbo')
        self.prompt_builder = PromptBuilder()
        
        # Set API key from config or environment
        api_key = self.llm_config.get('api_key')
        if api_key:
            openai.api_key = api_key
        
        logging.info(f"Initialized OpenAI connector with model: {self.model}")
    
    def generate_content(self, prompt: str) -> str:
        """Generate SQL content using OpenAI API
        
        Args:
            prompt: The prompt string for SQL generation
            
        Returns:
            Generated SQL query string
        """
        try:
            response = openai.ChatCompletion.create(
                model=self.model,
                messages=[
                    {"role": "user", "content": prompt}
                ],
                temperature=0.1  # Lower temperature for more deterministic SQL generation
            )
            
            return response.choices[0].message.content.strip()
        except Exception as e:
            logging.error(f"Error generating content from OpenAI: {str(e)}")
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