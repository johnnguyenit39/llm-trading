# README for CRUD API Author Implementation in Clean Architecture (Go)

## Overview

This project is built using **Clean Architecture** principles in Go, designed to help you quickly scaffold CRUD APIs for any object or table in your system. The goal is to keep the architecture modular and maintainable, while providing an easy way to implement API layers for different entities (tables/objects).

To streamline this process, a **mock folder** has been provided. The mock folder contains boilerplate code that can be easily customized to create CRUD operations for your tables. By following this simple process, you can implement the API for any object in just a few steps.

## Steps to Use the Author Folder

1. **Create Object / Table Definition**:
    - Start by defining the object or table for which you want to write a CRUD API (e.g., `Tag`, `User`, `Product`, etc.).

2. **Use Visual Studio Code Replace Feature**:
    - Open the mock folder in your code editor (e.g., Visual Studio Code).
    - Use the **Find and Replace** feature to replace all instances of `Author` with the name of your table/object (e.g., `Tag`).
    - This will automatically update all references to the mock object in the code.

3. **Update the File Name**:
    - After replacing the `Author` references in the code, update the filenames to match your object/table name. For example:
        - If you were working on a `Tag` object, rename files such as `mock_storage.go`, `mock_transport.go`, etc., to reflect the `Tag` object. 
        - This ensures your files are named according to the entity you are working with (e.g., `tag_storage.go`, `tag_transport.go`, etc.).

4. **Customize for Your Needs**:
    - After replacing the mock references, you can further customize the code if necessary to suit your specific business logic or storage requirements.

5. **Repeat for Each Object**:
    - You can repeat these steps for each object or table in your system by defining new objects and following the process for replacing `Author` and renaming files accordingly.

### Example: Creating CRUD API for Tag

1. **Define the `Tag` object**:  
    Create the `Tag` object with the necessary fields for your system (e.g., `ID`, `Name`).

2. **Replace `Author` with `Tag`**:  
    In Visual Studio Code, use the Find and Replace feature to replace every instance of `Author` with `Tag`.

3. **Rename Files**:  
    Rename files like `mock_storage.go` to `tag_storage.go`, `mock_transport.go` to `tag_transport.go`, and so on.

4. **Adjust Logic**:  
    Adjust the business logic, storage, and transport layers according to the needs of your `Tag` API.

By following this method, you can create a complete CRUD API for any object or table in just a few simple steps.

---

## Folder Structure

The mock folder contains the following layers:

- **Storage Layer**:  
    Contains the mock implementation for interacting with your data store (e.g., databases, file systems).

- **Business Layer**:  
    Contains the mock business logic for handling the entity's operations (e.g., validation, transformation).

- **Transport Layer**:  
    Contains the mock HTTP transport layer for defining the API routes and HTTP handlers.

Each layer is designed to be interchangeable and easy to extend. By following the replacement process, you only need to update the object/table name to customize the implementation for your specific use case.

---

## Benefits

- **Rapid API Creation**:  
    This approach allows you to quickly create a full set of CRUD APIs for any table or object in your system.

- **Maintainable Architecture**:  
    Following Clean Architecture principles ensures your code remains modular, maintainable, and scalable as your project grows.

- **Customizable**:  
    Once the boilerplate code is geneSubscriptiond, you can easily modify it to meet your specific business and application logic requirements.

---

## Example Folder Structure

