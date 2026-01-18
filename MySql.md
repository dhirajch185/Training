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
# Database Fundamentals & MySQL Overview

We use **Joins** to relate data between tables based on overlapping keys and shared data points.

---

## Best Practices: Data Normalization
Normalization is the process of organizing data to minimize redundancy and maximize integrity.
* **Atomicity:** Each row/column pair should contain a single, indivisible value.
* **Consistency:** Values should be directly relatable to the primary key of the table.

---

## MySQL Overview

### Benefits
* **Cost-Effective:** It is open-source and free to use.
* **User-Friendly:** Easy to set up and widely supported.
* **Open Source:** Highly customizable with a massive community.

### Disadvantages
* **Scalability:** Efficiency can decrease with extremely large databases.
* **Out-of-the-box Features:** Some advanced features require manual configuration or plugins.
* **Compliance:** Not 100% SQL compliant, which can complicate migrations to other DBMS.

---

## Data Types



### 1. Numeric Data
Used for mathematical operations.
* **Integer:** Whole numbers without decimals.
* **Fixed Point (DECIMAL):** Used when high precision is required (e.g., currency). Format: `-999.9`.
* **Floating Point:** Used when the number of significant figures is unpredictable.

### 2. String & Binary Data
* **CHAR:** Fixed length (max **255** characters). Storage is consistent regardless of data size because it is padded with spaces (spaces are hidden upon retrieval).
* **VARCHAR:** Variable length (up to **65,535** characters). Storage size adjusts dynamically based on the input.
* **BINARY / VARBINARY:** Similar to CHAR/VARCHAR but stores data as **bytes** rather than characters. Data is sorted numerically instead of alphabetically.
* **BLOB / TEXT:** For large data sets (up to 4GB).
    * **BLOB:** Stores data as binary (good for images/files).
    * **TEXT:** Stores data as character strings.

> **Note:** Use **CHAR/VARCHAR** for human-readable text and **BINARY/VARBINARY** for machine-readable data.

### 3. Date & Time
* **DATE:** Stores values as `YYYY-MM-DD`.
* **DATETIME:** Stores hours, minutes, seconds, and microseconds (`YYYY-MM-DD hh:mm:ss.fraction`).
* **TIMESTAMP:** Similar to DATETIME but adjusted to the **timezone**. Ideal for tracking "events" (like login times).
* **Usage:** Use DATE/DATETIME for static dates (birthdays) and TIMESTAMP for real-time events.

### 4. Special Types
* **BOOLEAN:** For `True` and `False` values.
* **SET / ENUM:** Defined during table creation for specific acceptable values.
    * **ENUM:** Allows only **one** valid value from the list.
    * **SET:** Allows for **multiple** valid values from the list.
* **BIT:** For storing binary values.
* **JSON:** Optimized storage for JSON formatted data.
* **Spatial Data:** For storing geographical/GIS data and map coordinates.

---
