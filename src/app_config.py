import os
import json
import logging
from typing import Any, Dict, Optional

class Config:
    """Configuration management for the AI Extension"""
    
    def __init__(self, config_file: Optional[str] = None):
        """Initialize configuration
        
        Args:
            config_file: Path to configuration file (optional)
        """
        self.config = {}
        
        # Default configuration
        self._set_defaults()
        
        # Load from config file if provided
        if config_file and os.path.exists(config_file):
            self._load_from_file(config_file)
        
        # Override with environment variables
        self._load_from_env()
        
        logging.debug(f"Configuration initialized: {self.config}")
    
    def _set_defaults(self):
        """Set default configuration values"""
        self.config = {
            "llm": {
                "provider": "ollama",
                "model": "llama3.2:1b",
                "server_url": "http://localhost:11434",
                "format": "json"
            },
            "grpc_server_address": "0.0.0.0:50051"
        }
    
    def _load_from_file(self, config_file: str):
        """Load configuration from file
        
        Args:
            config_file: Path to configuration file
        """
        try:
            with open(config_file, 'r') as f:
                file_config = json.load(f)
                self._update_nested_dict(self.config, file_config)
        except Exception as e:
            logging.error(f"Error loading config from file: {str(e)}")
    
    def _load_from_env(self):
        """Load configuration from environment variables"""
        # LLM configuration
        if os.getenv("OLLAMA_TEST_MODEL"):
            self.config["llm"]["model"] = os.getenv("OLLAMA_TEST_MODEL")
        
        if os.getenv("OLLAMA_SERVER_URL"):
            self.config["llm"]["server_url"] = os.getenv("OLLAMA_SERVER_URL")
        
        # Server configuration
        if os.getenv("GRPC_SERVER_ADDRESS"):
            self.config["grpc_server_address"] = os.getenv("GRPC_SERVER_ADDRESS")
    
    def _update_nested_dict(self, d: Dict, u: Dict):
        """Update nested dictionary recursively
        
        Args:
            d: Target dictionary to update
            u: Source dictionary with updates
        """
        for k, v in u.items():
            if isinstance(v, dict) and k in d and isinstance(d[k], dict):
                self._update_nested_dict(d[k], v)
            else:
                d[k] = v
    
    def get(self, key: str, default: Any = None) -> Any:
        """Get configuration value
        
        Args:
            key: Configuration key
            default: Default value if key not found
            
        Returns:
            Configuration value
        """
        return self.config.get(key, default)
    
    def get_llm_config(self) -> Dict[str, Any]:
        """Get LLM configuration
        
        Returns:
            LLM configuration dictionary
        """
        return self.config.get("llm", {})