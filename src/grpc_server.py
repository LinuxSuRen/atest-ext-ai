import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

import grpc
import logging
from concurrent import futures
from typing import Dict, Any

from ai_extension_pb2 import (
    GenerateContentRequest,
    GenerateContentResponse,
    ContentSuccessResponse,
    ErrorResponse,
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
    
    def GenerateContent(self, request: GenerateContentRequest, context) -> GenerateContentResponse:
        """Generate content based on natural language prompts with context
        
        Args:
            request: The GenerateContentRequest containing prompt and content type
            context: gRPC context
            
        Returns:
            GenerateContentResponse with either success or error result
        """
        try:
            logging.info(f"Received request - Content Type: {request.contentType}, Prompt: {request.prompt}")
            
            # Validate input
            if not request.prompt.strip():
                return GenerateContentResponse(
                    error=ErrorResponse(
                        code=ErrorCode.INVALID_ARGUMENT,
                        message="Prompt cannot be empty"
                    )
                )
            
            if not request.contentType.strip():
                return GenerateContentResponse(
                    error=ErrorResponse(
                        code=ErrorCode.INVALID_ARGUMENT,
                        message="Content type cannot be empty"
                    )
                )
            
            # Handle different content types
            if request.contentType.lower() == "sql":
                return self._generate_sql_content(request)
            elif request.contentType.lower() == "testcase":
                return self._generate_testcase_content(request)
            elif request.contentType.lower() == "mock":
                return self._generate_mock_content(request)
            else:
                return self._generate_generic_content(request)
                
        except Exception as e:
            logging.error(f"Error generating content: {str(e)}")
            return GenerateContentResponse(
                error=ErrorResponse(
                    code=ErrorCode.INTERNAL_ERROR,
                    message=f"Internal error: {str(e)}"
                )
            )
    
    def _generate_sql_content(self, request: GenerateContentRequest) -> GenerateContentResponse:
        """Generate SQL content from natural language"""
        try:
            # Extract SQL-specific parameters
            db_type = request.parameters.get('database_type', 'mysql')
            schemas = request.context.get('schemas', '').split(',') if request.context.get('schemas') else None
            
            # Build SQL generation prompt
            prompt = self.prompt_builder.build_sql_generation_prompt(
                natural_language_input=request.prompt,
                database_type=db_type,
                schemas=schemas,
                examples=None  # Could be extracted from parameters if needed
            )
            
            # Generate SQL using LLM
            messages = [
                {"role": "system", "content": "You are an AI assistant that translates natural language to SQL queries."},
                {"role": "user", "content": prompt}
            ]
            
            response = self.llm_connector.generate_content(messages)
            
            if not response or 'content' not in response:
                return GenerateContentResponse(
                    error=ErrorResponse(
                        code=ErrorCode.TRANSLATION_FAILED,
                        message="Failed to generate SQL from LLM"
                    )
                )
            
            sql_query = response['content'].strip()
            
            # Post-process SQL for dialect-specific adjustments
            sql_query = self.dialect_manager.adapt_sql_for_dialect(sql_query, db_type)
            
            # Create success response
            success_response = ContentSuccessResponse(
                content=sql_query,
                explanation=f"Generated {db_type} SQL query from natural language input",
                confidenceScore=response.get('confidence', 0.8),
                metadata={
                    "content_type": "sql",
                    "database_type": db_type
                }
            )
            
            logging.info(f"Generated SQL: {sql_query}")
            
            return GenerateContentResponse(success=success_response)
            
        except Exception as e:
            logging.error(f"Error generating SQL content: {str(e)}")
            return GenerateContentResponse(
                error=ErrorResponse(
                    code=ErrorCode.INTERNAL_ERROR,
                    message=f"Internal error: {str(e)}"
                )
            )
    
    def _generate_testcase_content(self, request: GenerateContentRequest) -> GenerateContentResponse:
        """Generate test case content"""
        try:
            # Build test case generation prompt
            prompt = f"Generate test cases for: {request.prompt}"
            
            # Add context if available
            if request.context:
                context_info = ", ".join([f"{k}: {v}" for k, v in request.context.items()])
                prompt += f"\nContext: {context_info}"
            
            messages = [
                {"role": "system", "content": "You are an AI assistant that generates comprehensive test cases."},
                {"role": "user", "content": prompt}
            ]
            
            response = self.llm_connector.generate_content(messages)
            
            if not response or 'content' not in response:
                return GenerateContentResponse(
                    error=ErrorResponse(
                        code=ErrorCode.TRANSLATION_FAILED,
                        message="Failed to generate test cases from LLM"
                    )
                )
            
            success_response = ContentSuccessResponse(
                content=response['content'].strip(),
                explanation="Generated test cases based on the provided requirements",
                confidenceScore=response.get('confidence', 0.8),
                metadata={
                    "content_type": "testcase"
                }
            )
            
            return GenerateContentResponse(success=success_response)
            
        except Exception as e:
            logging.error(f"Error generating test case content: {str(e)}")
            return GenerateContentResponse(
                error=ErrorResponse(
                    code=ErrorCode.INTERNAL_ERROR,
                    message=f"Internal error: {str(e)}"
                )
            )
    
    def _generate_mock_content(self, request: GenerateContentRequest) -> GenerateContentResponse:
        """Generate mock service content"""
        try:
            # Build mock service generation prompt
            prompt = f"Generate mock service for: {request.prompt}"
            
            # Add context if available
            if request.context:
                context_info = ", ".join([f"{k}: {v}" for k, v in request.context.items()])
                prompt += f"\nContext: {context_info}"
            
            messages = [
                {"role": "system", "content": "You are an AI assistant that generates mock services and API responses."},
                {"role": "user", "content": prompt}
            ]
            
            response = self.llm_connector.generate_content(messages)
            
            if not response or 'content' not in response:
                return GenerateContentResponse(
                    error=ErrorResponse(
                        code=ErrorCode.TRANSLATION_FAILED,
                        message="Failed to generate mock service from LLM"
                    )
                )
            
            success_response = ContentSuccessResponse(
                content=response['content'].strip(),
                explanation="Generated mock service based on the provided requirements",
                confidenceScore=response.get('confidence', 0.8),
                metadata={
                    "content_type": "mock"
                }
            )
            
            return GenerateContentResponse(success=success_response)
            
        except Exception as e:
            logging.error(f"Error generating mock content: {str(e)}")
            return GenerateContentResponse(
                error=ErrorResponse(
                    code=ErrorCode.INTERNAL_ERROR,
                    message=f"Internal error: {str(e)}"
                )
            )
    
    def _generate_generic_content(self, request: GenerateContentRequest) -> GenerateContentResponse:
        """Generate generic content for unknown content types"""
        try:
            # Build generic generation prompt
            prompt = f"Generate {request.contentType} content for: {request.prompt}"
            
            # Add context if available
            if request.context:
                context_info = ", ".join([f"{k}: {v}" for k, v in request.context.items()])
                prompt += f"\nContext: {context_info}"
            
            messages = [
                {"role": "system", "content": f"You are an AI assistant that generates {request.contentType} content."},
                {"role": "user", "content": prompt}
            ]
            
            response = self.llm_connector.generate_content(messages)
            
            if not response or 'content' not in response:
                return GenerateContentResponse(
                    error=ErrorResponse(
                        code=ErrorCode.TRANSLATION_FAILED,
                        message=f"Failed to generate {request.contentType} content from LLM"
                    )
                )
            
            success_response = ContentSuccessResponse(
                content=response['content'].strip(),
                explanation=f"Generated {request.contentType} content based on the provided requirements",
                confidenceScore=response.get('confidence', 0.8),
                metadata={
                    "content_type": request.contentType
                }
            )
            
            return GenerateContentResponse(success=success_response)
            
        except Exception as e:
            logging.error(f"Error generating generic content: {str(e)}")
            return GenerateContentResponse(
                error=ErrorResponse(
                    code=ErrorCode.INTERNAL_ERROR,
                    message=f"Internal error: {str(e)}"
                )
            )
    



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