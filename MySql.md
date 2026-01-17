# Learning MySQL Development - Course Notes

Detailed notes and concepts from the [Learning MySQL Development](https://www.linkedin.com/learning-login/share?account=2154233&forceAccount=false&redirect=https%3A%2F%2Fwww.linkedin.com%2Flearning%2Flearning-mysql-development-22651630%3Ftrk%3Dshare_ent_url%26shareId%3DbYjWUj7PTRyg2PD%252FFsujwg%253D%253D) course on LinkedIn Learning.

---

## 🏗️ DBMS (Database Management System)
A DBMS is a service that provides the structure and functionality to interact with data. It manages how data is stored, retrieved, and secured.

* **Capabilities:** Provides CRUD (Create, Read, Update, Delete) operations.
* **Security:** Enforces user restrictions for reading and writing data.
* **SQL:** The primary language for interaction.
* **Declarative:** SQL describes **what** result is wanted, not the step-by-step program flow.

### Core Data Definition Commands
* `CREATE TABLE`: Add new tables to the schema.
* `ALTER TABLE`: Modify existing table structures.
* `TRUNCATE`: Remove all data from a table while keeping the structure.
* `DROP`: Delete a table entirely.

> [!CAUTION]
> In MySQL, DDL operations (like DROP or TRUNCATE) typically cannot be undone. There is no "undo" button once a table is dropped.

---

## 📈 Database vs. Flat Files
Databases offer significant advantages over spreadsheets and flat files:

| Feature | Description |
| :--- | :--- |
| **Flexibility** | Ability to pull data from multiple tables and join them by interrelated fields. |
| **Scalability** | Designed to handle large volumes of data that flat files cannot support. |
| **Integrity** | Ensures data consistency and prevents one user's actions from corrupting data for others. |

---

## 📖 Key Terminology
In a **Relational Database**, data is organized into tables where columns represent specific entities (text, numbers, etc.).

| Standard Term | Formal/Relational Term |
| :--- | :--- |
| Table | Relation |
| Row | Record / Tuple |
| Column | Attribute / Field |

---

## ⌨️ SQL Syntax & Statements
Users interact with the database using **SQL Statements**, which always end with a semicolon (`;`).

### CRUD Operations
* **Create:** `INSERT`
* **Read:** `SELECT` (also known as a **Query**)
* **Update:** `UPDATE`
* **Delete:** `DELETE`

### Example Query
```sql
SELECT * FROM table_name 
WHERE column_name > 10 
ORDER BY column_name ASC;

---
We can use joins to relate data between tables based on overlapping data.
---
Best Practices
- Data normalization - design to min database redudancy and max integrity
                      - Each row/column pair should have a single value
                      - These values should be relatable to key values of the table
---
MySql Benefits
  - Its free. ( No official tech support)
  - Its easy to use
  - Its opensource
Disadvantages
  - with huge databases its less efficient
  - Some features are not avaialble out of the box
  - Not fully sql compliant for migration to other DBMS
---
Data Types
  -  Numeric data (numbers for math)- Integer, fixed point(Decimal.scale format eg -999.9),floating point (decimal floats to allow more decimal values)
  -  String (for Text)
  -  
