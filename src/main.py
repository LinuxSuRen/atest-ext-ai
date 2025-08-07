#!/usr/bin/env python3

import os
import argparse
import logging
from src.grpc_server import serve
from src.config import Config

def main():
    """Main entry point for the AI Extension service"""
    parser = argparse.ArgumentParser(description="AI Extension for atest")
    parser.add_argument("-v", "--verbose", action="store_true", help="verbose mode")
    args = parser.parse_args()
    
    # Configure logging
    log_level = logging.DEBUG if args.verbose else logging.INFO
    logging.basicConfig(level=log_level, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
    
    # Load configuration
    config = Config()
    
    # Start gRPC server
    serve(config)

if __name__ == "__main__":
    main()