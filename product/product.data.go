package product

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/g0ng0n-dev/webservicito/database"
	"log"
	"strings"
	"sync"
	"time"
)

/* We are using a mutex because our Webservices are multithreaded and maps and go are inherently not thread safe.
 * which means we need to wrap our map using a mutex to avoid 2 threads from writing and reading the map at the same time.
*/
var productMap = struct {
	sync.RWMutex
	m map[int]Product

}{m: make(map[int]Product)}

func getProduct(productID int) (*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	row := database.DbConn.QueryRowContext(ctx,` SELECT productId, 
	manufacturer, 
	sku, 
	upc, 
	pricePerUnit, 
	quantityOnHand,
	productName
	FROM inventorydb.products
	WHERE productId = ?; `, productID)

	product := &Product{}

	err := row.Scan(&product.ProductID,
		&product.Manufacturer,
		&product.Sku,
		&product.Upc,
		&product.PricePerUnit,
		&product.QuantityOnHand,
		&product.ProductName)

	if err == sql.ErrNoRows {
		return nil, nil
	}else if err != nil {
		return nil, err
	}
	return product, nil
}

func removeProduct(productID int) error{
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := database.DbConn.ExecContext(ctx,`DELETE FROM inventorydb.products WHERE productId = ?`, productID)
	if err != nil {
		return err
	}
	return nil
}

func getProductList() ([]Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	results, err := database.DbConn.QueryContext(ctx,`SELECT productId, 
	manufacturer, 
	sku, 
	upc, 
	pricePerUnit, 
	quantityOnHand,
	productName
	FROM inventorydb.products;`)

	if err != nil {
		fmt.Println("Error On DB", err)
		return nil, err
	}

	defer results.Close()
	products := make([]Product, 0)

	for results.Next() {
		var product Product
		results.Scan(&product.ProductID,
			&product.Manufacturer,
			&product.Sku,
			&product.Upc,
			&product.PricePerUnit,
			&product.QuantityOnHand,
			&product.ProductName)
		products = append(products, product)
	}

	return products, nil
}


func updateProduct(product Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := database.DbConn.ExecContext(ctx, `UPDATE products SET 
		manufacturer=?,
		sku=?,
		upc=?,
		pricePerUnit=CAST(? AS DECIMAL (13,2)),
		quantityOnHand=?,
		productName=?
		WHERE productId=?`,
		product.Manufacturer,
		product.Sku,
		product.Upc,
		product.PricePerUnit,
		product.QuantityOnHand,
		product.ProductName,
		product.ProductID)
	if err != nil {
		return err
	}
	return nil

}

func insertProduct(product Product) (int, error){
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, err := database.DbConn.ExecContext(ctx,`INSERT INTO products
	(manufacturer,
		sku,
		upc,
		pricePerUnit,
		quantityOnHand,
		productName) VALUES (?,?,?,?,?,?)`,
		product.Manufacturer,
		product.Sku,
		product.Upc,
		product.PricePerUnit,
		product.QuantityOnHand,
		product.ProductName)

	if err != nil {
		return 0, nil
	}
	insertID, err := result.LastInsertId()
	if err != nil {
		return 0, nil
	}
	return int(insertID), nil
}

func GetTopTenProducts() ([]Product, error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, err := database.DbConn.QueryContext(ctx,`SELECT productId, 
	manufacturer, 
	sku, 
	upc, 
	pricePerUnit, 
	quantityOnHand,
	productName
	FROM inventorydb.products ORDER BY quantityOnHand DESC LIMIT 10;`)

	if err != nil {
		fmt.Println("Error On DB", err)
		return nil, err
	}

	defer results.Close()
	products := make([]Product, 0)

	for results.Next() {
		var product Product
		results.Scan(&product.ProductID,
			&product.Manufacturer,
			&product.Sku,
			&product.Upc,
			&product.PricePerUnit,
			&product.QuantityOnHand,
			&product.ProductName)
		products = append(products, product)
	}

	return products, nil
}

func SearchForProductData(productFilter ProductReportFilter) ([]Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var queryArgs = make([]interface{},0)
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`SELECT
		productId,
		LOWER(manufacturer),
		LOWER(sku),
		upc,
		pricePerUnit,
		quantityOnHand,
		LOWER(productName)
		FROM inventorydb.products WHERE`)
	if productFilter.NameFilter != ""{
		queryBuilder.WriteString(`productName LIKE ?`)
		queryArgs = append(queryArgs, "%"+strings.ToLower(productFilter.NameFilter) + "%")
	}
	if productFilter.ManufacturerFilter != "" {
		if len(queryArgs)>0{
			queryBuilder.WriteString(" AND ")
		}
		queryBuilder.WriteString(`manufacturer LIKE ?`)
		queryArgs = append(queryArgs, "%"+strings.ToLower(productFilter.ManufacturerFilter) + "%")
	}
	if productFilter.SKUFILTER != "" {
		if len(queryArgs)>0{
			queryBuilder.WriteString(" AND ")
		}
		queryBuilder.WriteString(`sku LIKE ?`)
		queryArgs = append(queryArgs, "%"+strings.ToLower(productFilter.SKUFILTER) + "%")
	}

	results, err := database.DbConn.QueryContext(ctx, queryBuilder.String(), queryArgs...)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer results.Close()

	products := make([]Product,0)
	for results.Next() {
		var product Product
		results.Scan(&product.ProductID,
			&product.Manufacturer,
			&product.Sku,
			&product.Upc,
			&product.PricePerUnit,
			&product.QuantityOnHand,
			&product.ProductName)

		products = append(products, product)
	}

	return products, nil

}