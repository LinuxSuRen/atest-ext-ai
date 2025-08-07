import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

import grpc
import logging
from concurrent import futures
from typing import Dict, Any

from ai_extension_pb2 import (
    GenerateSQLRequest,
    GenerateSQLResponse,
    SuccessResponse,
    ErrorResponse,
    DatabaseType,
    ErrorCode
)
from ai_extension_pb2_grpc import (
    AIExtensionServicer,
    add_AIExtensionServicer_to_server
)
from app_config import Config
from llm.ollama import OllamaConnector
from llm.openai import OpenAIConnector
from llm.prompt_builder import PromptBuilder
from db.dialect_manager import DialectManager

class AIExtensionService(AIExtensionServicer):
    """Implementation of the AIExtension gRPC service"""
    
    def __init__(self):
        """Initialize the AI Extension service"""
        self.config = Config()
        self.llm_config = self.config.get_llm_config()
        self.prompt_builder = PromptBuilder()
        self.dialect_manager = DialectManager()
        
        # Initialize LLM connector based on configuration
        self._init_llm_connector()
        
        logging.info(f"AIExtensionService initialized with provider: {self.llm_config.get('provider')}")
    
    def _init_llm_connector(self):
        """Initialize the appropriate LLM connector based on configuration"""
        provider = self.llm_config.get('provider', 'ollama')
        
        if provider == 'ollama':
            self.llm_connector = OllamaConnector(self.llm_config)
        elif provider == 'openai':
            self.llm_connector = OpenAIConnector(self.llm_config)
        else:
            raise ValueError(f"Unsupported LLM provider: {provider}")
    
    def GenerateSQLFromNaturalLanguage(self, request: GenerateSQLRequest, context) -> GenerateSQLResponse:
        """Generate SQL query from natural language input
        
        Args:
            request: The GenerateSQLRequest containing natural language input
            context: gRPC context
            
        Returns:
            GenerateSQLResponse with either success or error result
        """
        try:
            logging.info(f"Received request: {request.natural_language_input}")
            
            # Validate input
            if not request.natural_language_input.strip():
                return GenerateSQLResponse(
                    error=ErrorResponse(
                        code=ErrorCode.INVALID_ARGUMENT,
                        message="Natural language input cannot be empty"
                    )
                )
            
            # Get database type string
            db_type = self._get_database_type_string(request.database_type)
            if not db_type:
                return GenerateSQLResponse(
                    error=ErrorResponse(
                        code=ErrorCode.UNSUPPORTED_DATABASE,
                        message=f"Unsupported database type: {request.database_type}"
                    )
                )
            
            # Build prompt
            prompt = self.prompt_builder.build_sql_generation_prompt(
                natural_language_input=request.natural_language_input,
                database_type=db_type,
                schemas=list(request.schemas) if request.schemas else None,
                examples=[
                    {"natural_language": ex.natural_language_prompt, "sql": ex.sql_query}
                    for ex in request.examples
                ] if request.examples else None
            )
            
            # Generate SQL using LLM
            messages = [
                {"role": "system", "content": "You are an AI assistant that translates natural language to SQL queries."},
                {"role": "user", "content": prompt}
            ]
            
            response = self.llm_connector.generate_content(messages)
            
            if not response or 'content' not in response:
                return GenerateSQLResponse(
                    error=ErrorResponse(
                        code=ErrorCode.TRANSLATION_FAILED,
                        message="Failed to generate SQL from LLM"
                    )
                )
            
            sql_query = response['content'].strip()
            
            # Post-process SQL for dialect-specific adjustments
            sql_query = self.dialect_manager.adapt_sql_for_dialect(sql_query, db_type)
            
            # Create success response
            success_response = SuccessResponse(
                sql_query=sql_query,
                explanation=f"Generated {db_type} SQL query from natural language input",
                confidence_score=response.get('confidence', 0.8)
            )
            
            logging.info(f"Generated SQL: {sql_query}")
            
            return GenerateSQLResponse(success=success_response)
            
        except Exception as e:
            logging.error(f"Error generating SQL: {str(e)}")
            return GenerateSQLResponse(
                error=ErrorResponse(
                    code=ErrorCode.INTERNAL_ERROR,
                    message=f"Internal error: {str(e)}"
                )
            )
    
    def _get_database_type_string(self, db_type: int) -> str:
        """Convert DatabaseType enum to string
        
        Args:
            db_type: DatabaseType enum value
            
        Returns:
            Database type string or None if unsupported
        """
        type_mapping = {
            DatabaseType.MYSQL: "mysql",
            DatabaseType.POSTGRESQL: "postgresql",
            DatabaseType.SQLITE: "sqlite"
        }
        return type_mapping.get(db_type)


def serve():
    """Start the gRPC server"""
    # Initialize configuration
    config = Config()
    
    # Create server
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    
    # Add the AI Extension service
    ai_service = AIExtensionService()
    add_AIExtensionServicer_to_server(ai_service, server)
    
    # Configure server address
    server_address = config.get("grpc_server_address", "0.0.0.0:50051")
    server.add_insecure_port(server_address)
    
    # Start server
    server.start()
    print(f"AI Extension gRPC server started on {server_address}")
    
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        print("\nShutting down server...")
        server.stop(0)


if __name__ == "__main__":
    # Start the server
    serve()