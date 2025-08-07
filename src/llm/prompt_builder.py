from typing import List, Dict, Any, Optional

class PromptBuilder:
    """Builder for constructing prompts for SQL generation"""
    
    def __init__(self):
        """Initialize the prompt builder"""
        pass
    
    def build_sql_generation_prompt(
        self,
        natural_language_input: str,
        database_type: str,
        schemas: Optional[List[str]] = None,
        examples: Optional[List[Dict[str, str]]] = None
    ) -> str:
        """Build a prompt for SQL generation from natural language
        
        Args:
            natural_language_input: The user's natural language query
            database_type: Target database type (mysql, postgresql, sqlite)
            schemas: Optional list of DDL statements for table schemas
            examples: Optional list of example natural language to SQL pairs
            
        Returns:
            Formatted prompt string for the LLM
        """
        prompt_parts = []
        
        # Base instruction
        prompt_parts.append(
            f"You are an AI assistant that translates human-readable requests into {database_type.upper()} SQL queries."
        )
        
        # Add schema information if provided
        if schemas:
            prompt_parts.append("\nGiven the following database schema:")
            for schema in schemas:
                prompt_parts.append(f"\n{schema}")
        
        # Add examples if provided
        if examples:
            prompt_parts.append("\nHere are some examples of natural language to SQL translations:")
            for i, example in enumerate(examples, 1):
                prompt_parts.append(
                    f"\nExample {i}:\n"
                    f"Natural Language: {example['natural_language']}\n"
                    f"SQL: {example['sql']}"
                )
        
        # Add the main request
        prompt_parts.append(f"\nTranslate this request: \"{natural_language_input}\"")
        
        # Add response format instruction
        prompt_parts.append(
            "\nRespond ONLY with the SQL query. Do not include any explanations, "
            "markdown formatting, or additional text. The response should be a valid "
            f"{database_type.upper()} SQL statement that can be executed directly."
        )
        
        return "".join(prompt_parts)
    
    def build_schema_aware_prompt(
        self,
        natural_language_input: str,
        database_type: str,
        table_schemas: Dict[str, List[str]]
    ) -> str:
        """Build a schema-aware prompt with detailed table information
        
        Args:
            natural_language_input: The user's natural language query
            database_type: Target database type
            table_schemas: Dictionary mapping table names to their column definitions
            
        Returns:
            Formatted prompt string with detailed schema information
        """
        prompt_parts = []
        
        prompt_parts.append(
            f"You are an expert {database_type.upper()} SQL query generator. "
            "Generate accurate SQL queries based on natural language requests."
        )
        
        # Add detailed schema information
        if table_schemas:
            prompt_parts.append("\nDatabase Schema:")
            for table_name, columns in table_schemas.items():
                prompt_parts.append(f"\nTable: {table_name}")
                prompt_parts.append("Columns:")
                for column in columns:
                    prompt_parts.append(f"  - {column}")
        
        # Add the request
        prompt_parts.append(f"\nUser Request: {natural_language_input}")
        
        # Add specific instructions
        prompt_parts.append(
            "\nInstructions:\n"
            "1. Generate a valid SQL query that fulfills the user's request\n"
            "2. Use only the tables and columns defined in the schema above\n"
            "3. Follow proper SQL syntax and best practices\n"
            f"4. Ensure compatibility with {database_type.upper()}\n"
            "5. Return ONLY the SQL query without any explanations or formatting"
        )
        
        return "".join(prompt_parts)
    
    def build_conversational_prompt(
        self,
        natural_language_input: str,
        database_type: str,
        conversation_history: List[Dict[str, str]],
        schemas: Optional[List[str]] = None
    ) -> str:
        """Build a prompt that takes conversation history into account
        
        Args:
            natural_language_input: Current user input
            database_type: Target database type
            conversation_history: Previous exchanges in the conversation
            schemas: Optional schema information
            
        Returns:
            Conversational prompt string
        """
        prompt_parts = []
        
        prompt_parts.append(
            f"You are an AI assistant helping with {database_type.upper()} SQL query generation. "
            "You maintain context from previous interactions in this conversation."
        )
        
        # Add schema if available
        if schemas:
            prompt_parts.append("\nDatabase Schema:")
            for schema in schemas:
                prompt_parts.append(f"\n{schema}")
        
        # Add conversation history
        if conversation_history:
            prompt_parts.append("\nConversation History:")
            for exchange in conversation_history:
                prompt_parts.append(
                    f"\nUser: {exchange.get('user', '')}\n"
                    f"Assistant: {exchange.get('assistant', '')}"
                )
        
        # Add current request
        prompt_parts.append(f"\nCurrent Request: {natural_language_input}")
        
        # Add instructions
        prompt_parts.append(
            "\nGenerate a SQL query that addresses the current request, "
            "taking into account the conversation context. "
            "Respond with ONLY the SQL query."
        )
        
        return "".join(prompt_parts)