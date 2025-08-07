import pytest
from unittest.mock import Mock, patch

from src.llm.prompt_builder import PromptBuilder
from src.llm.ollama import OllamaConnector
from src.llm.openai import OpenAIConnector
from src.db.dialect_manager import DialectManager
from src.utils.helpers import clean_sql_query, validate_sql_syntax, extract_table_names

class TestPromptBuilder:
    """Test cases for PromptBuilder class"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.prompt_builder = PromptBuilder()
    
    def test_build_sql_generation_prompt_basic(self):
        """Test basic SQL generation prompt building"""
        prompt = self.prompt_builder.build_sql_generation_prompt(
            natural_language_input="Show me all users",
            database_type="mysql"
        )
        
        assert "MYSQL" in prompt
        assert "Show me all users" in prompt
        assert "Respond ONLY with the SQL query" in prompt
    
    def test_build_sql_generation_prompt_with_schema(self):
        """Test SQL generation prompt with schema information"""
        schemas = ["CREATE TABLE users (id INT, name VARCHAR(100))"]
        
        prompt = self.prompt_builder.build_sql_generation_prompt(
            natural_language_input="Get all user names",
            database_type="postgresql",
            schemas=schemas
        )
        
        assert "POSTGRESQL" in prompt
        assert "CREATE TABLE users" in prompt
        assert "Get all user names" in prompt
    
    def test_build_sql_generation_prompt_with_examples(self):
        """Test SQL generation prompt with examples"""
        examples = [
            {
                "natural_language": "Show all products",
                "sql": "SELECT * FROM products;"
            }
        ]
        
        prompt = self.prompt_builder.build_sql_generation_prompt(
            natural_language_input="List all customers",
            database_type="sqlite",
            examples=examples
        )
        
        assert "Show all products" in prompt
        assert "SELECT * FROM products" in prompt
        assert "List all customers" in prompt

class TestDialectManager:
    """Test cases for DialectManager class"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.dialect_manager = DialectManager()
    
    def test_mysql_adjustments(self):
        """Test MySQL-specific SQL adjustments"""
        sql = 'SELECT "column_name" FROM table WHERE flag = TRUE;'
        adjusted = self.dialect_manager.adjust_sql_for_dialect(sql, "mysql")
        
        assert "`column_name`" in adjusted
        assert "= 1" in adjusted
    
    def test_postgresql_adjustments(self):
        """Test PostgreSQL-specific SQL adjustments"""
        sql = "SELECT `column_name` FROM table WHERE flag = 1;"
        adjusted = self.dialect_manager.adjust_sql_for_dialect(sql, "postgresql")
        
        assert '"column_name"' in adjusted
    
    def test_sqlite_adjustments(self):
        """Test SQLite-specific SQL adjustments"""
        sql = "SELECT * FROM users LIMIT 10;"
        adjusted = self.dialect_manager.adjust_sql_for_dialect(sql, "sqlite")
        
        # SQLite should return the original SQL with minimal changes
        assert "SELECT * FROM users LIMIT 10" in adjusted
    
    def test_unsupported_database(self):
        """Test handling of unsupported database types"""
        sql = "SELECT * FROM users;"
        adjusted = self.dialect_manager.adjust_sql_for_dialect(sql, "unknown_db")
        
        # Should return original SQL for unknown database types
        assert adjusted == sql
    
    def test_get_supported_databases(self):
        """Test getting list of supported databases"""
        supported = self.dialect_manager.get_supported_databases()
        
        assert "mysql" in supported
        assert "postgresql" in supported
        assert "sqlite" in supported
        assert "mssql" in supported
        assert "oracle" in supported
    
    def test_validate_database_type(self):
        """Test database type validation"""
        assert self.dialect_manager.validate_database_type("mysql") is True
        assert self.dialect_manager.validate_database_type("POSTGRESQL") is True
        assert self.dialect_manager.validate_database_type("unknown") is False

class TestHelpers:
    """Test cases for helper functions"""
    
    def test_clean_sql_query(self):
        """Test SQL query cleaning"""
        # Test markdown removal
        sql_with_markdown = "```sql\nSELECT * FROM users\n```"
        cleaned = clean_sql_query(sql_with_markdown)
        assert cleaned == "SELECT * FROM users;"
        
        # Test whitespace normalization
        sql_with_spaces = "SELECT   *    FROM   users"
        cleaned = clean_sql_query(sql_with_spaces)
        assert cleaned == "SELECT * FROM users;"
        
        # Test semicolon addition
        sql_without_semicolon = "SELECT * FROM users"
        cleaned = clean_sql_query(sql_without_semicolon)
        assert cleaned.endswith(";")
    
    def test_validate_sql_syntax(self):
        """Test basic SQL syntax validation"""
        # Valid SQL
        assert validate_sql_syntax("SELECT * FROM users;") is True
        assert validate_sql_syntax("INSERT INTO users (name) VALUES ('John');") is True
        
        # Invalid SQL
        assert validate_sql_syntax("") is False
        assert validate_sql_syntax("INVALID QUERY") is False
        assert validate_sql_syntax("SELECT * FROM users (") is False  # Unmatched parentheses
    
    def test_extract_table_names(self):
        """Test table name extraction from SQL"""
        sql = "SELECT u.name FROM users u JOIN orders o ON u.id = o.user_id"
        tables = extract_table_names(sql)
        
        assert "USERS" in tables
        assert "ORDERS" in tables
        
        # Test with quoted table names
        sql_quoted = 'SELECT * FROM "user_table" JOIN `order_table`'
        tables_quoted = extract_table_names(sql_quoted)
        
        assert "USER_TABLE" in tables_quoted
        assert "ORDER_TABLE" in tables_quoted

class TestLLMConnectors:
    """Test cases for LLM connector classes"""
    
    @patch('src.llm.ollama.requests.post')
    def test_ollama_connector_generate_content(self, mock_post):
        """Test Ollama connector content generation"""
        # Mock response
        mock_response = Mock()
        mock_response.json.return_value = {"response": "SELECT * FROM users;"}
        mock_response.raise_for_status.return_value = None
        mock_post.return_value = mock_response
        
        # Mock config
        config = Mock()
        config.get_llm_config.return_value = {
            "model": "llama3.2:1b",
            "server_url": "http://localhost:11434"
        }
        
        connector = OllamaConnector(config)
        result = connector.generate_content("Show me all users")
        
        assert result == "SELECT * FROM users;"
        mock_post.assert_called_once()
    
    @patch('src.llm.openai.openai.ChatCompletion.create')
    def test_openai_connector_generate_content(self, mock_create):
        """Test OpenAI connector content generation"""
        # Mock response
        mock_response = Mock()
        mock_response.choices = [Mock()]
        mock_response.choices[0].message.content = "SELECT * FROM users;"
        mock_create.return_value = mock_response
        
        # Mock config
        config = Mock()
        config.get_llm_config.return_value = {
            "model": "gpt-3.5-turbo",
            "api_key": "test-key"
        }
        
        connector = OpenAIConnector(config)
        result = connector.generate_content("Show me all users")
        
        assert result == "SELECT * FROM users;"
        mock_create.assert_called_once()
    
    def test_ollama_generate_sql(self):
        """Test Ollama SQL generation method"""
        config = Mock()
        config.get_llm_config.return_value = {
            "model": "llama3.2:1b",
            "server_url": "http://localhost:11434"
        }
        
        connector = OllamaConnector(config)
        
        # Mock the generate_content method
        with patch.object(connector, 'generate_content', return_value="SELECT * FROM users;"):
            result = connector.generate_sql(
                natural_language_input="Show all users",
                database_type="mysql"
            )
            
            assert result == "SELECT * FROM users;"
    
    def test_openai_generate_sql(self):
        """Test OpenAI SQL generation method"""
        config = Mock()
        config.get_llm_config.return_value = {
            "model": "gpt-3.5-turbo",
            "api_key": "test-key"
        }
        
        connector = OpenAIConnector(config)
        
        # Mock the generate_content method
        with patch.object(connector, 'generate_content', return_value="SELECT * FROM users;"):
            result = connector.generate_sql(
                natural_language_input="Show all users",
                database_type="postgresql"
            )
            
            assert result == "SELECT * FROM users;"

if __name__ == "__main__":
    pytest.main([__file__])