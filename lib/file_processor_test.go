package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMoveFileToSuccess_Success(t *testing.T) {
	// 1. Setup
	srcDir := t.TempDir()  // テスト用ソースディレクトリ
	destDir := t.TempDir() // テスト用移動先ディレクトリ
	srcFileName := "test_success.xlsx"
	srcFilePath := filepath.Join(srcDir, srcFileName)
	expectedDestPath := filepath.Join(destDir, srcFileName)

	// ソースファイル作成
	f, err := os.Create(srcFilePath)
	if err != nil {
		t.Fatalf("テスト用ソースファイル作成失敗: %v", err)
	}
	f.Close()

	// 2. Execute
	err = moveFileToSuccess(srcFilePath, destDir)

	// 3. Assert
	if err != nil {
		t.Fatalf("moveFileToSuccess で予期せぬエラー: %v", err)
	}

	// 移動元ファイルがなくなったことを確認
	if _, err := os.Stat(srcFilePath); err == nil || !os.IsNotExist(err) {
		t.Errorf("移動後もソースファイルが存在します: %s", srcFilePath)
	}

	// 移動先にファイルが作成されたことを確認
	if _, err := os.Stat(expectedDestPath); os.IsNotExist(err) {
		t.Errorf("移動先にファイルが作成されませんでした: %s", expectedDestPath)
	}
}

func TestMoveFileToSuccess_DestDirNotExist(t *testing.T) {
	// 1. Setup
	srcDir := t.TempDir()
	destDir := filepath.Join(t.TempDir(), "non_existent_subdir") // 存在しないパス
	srcFileName := "test_dest_not_exist.xlsx"
	srcFilePath := filepath.Join(srcDir, srcFileName)

	// ソースファイル作成
	f, err := os.Create(srcFilePath)
	if err != nil {
		t.Fatalf("テスト用ソースファイル作成失敗: %v", err)
	}
	f.Close()

	// 2. Execute
	err = moveFileToSuccess(srcFilePath, destDir)

	// 3. Assert
	if err == nil {
		t.Fatal("移動先ディレクトリが存在しない場合にエラーが返されませんでした")
	}
	expectedErrMsg := fmt.Sprintf("移動先ディレクトリ '%s' が存在しません", destDir)
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("期待されるエラーメッセージが含まれていません。\n期待含む: %s\n実際: %v", expectedErrMsg, err)
	}
	// 移動元ファイルが残っていることを確認
	if _, errStat := os.Stat(srcFilePath); os.IsNotExist(errStat) {
		t.Errorf("移動先なしエラー時にソースファイルが削除されました: %s", srcFilePath)
	}
}

func TestMoveFileToSuccess_DestIsNotDir(t *testing.T) {
	// 1. Setup
	srcDir := t.TempDir()
	// 移動先としてファイルパスを指定
	destFilePath := filepath.Join(t.TempDir(), "destination_is_a_file.txt")
	srcFileName := "test_dest_is_file.xlsx"
	srcFilePath := filepath.Join(srcDir, srcFileName)

	// ソースファイル作成
	fSrc, err := os.Create(srcFilePath)
	if err != nil {
		t.Fatal(err)
	}
	fSrc.Close()
	// 移動先ファイル作成
	fDest, err := os.Create(destFilePath)
	if err != nil {
		t.Fatal(err)
	}
	fDest.Close()

	// 2. Execute
	err = moveFileToSuccess(srcFilePath, destFilePath) // destDir にファイルのパスを渡す

	// 3. Assert
	if err == nil {
		t.Fatal("移動先がディレクトリでない場合にエラーが返されませんでした")
	}
	expectedErrMsg := fmt.Sprintf("移動先パス '%s' はディレクトリではありません", destFilePath)
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("期待されるエラーメッセージが含まれていません。\n期待含む: %s\n実際: %v", expectedErrMsg, err)
	}
	// 移動元ファイルが残っていることを確認
	if _, errStat := os.Stat(srcFilePath); os.IsNotExist(errStat) {
		t.Errorf("移動先不正エラー時にソースファイルが削除されました: %s", srcFilePath)
	}
}

func TestMoveFileToSuccess_DestFileExists(t *testing.T) {
	// 1. Setup
	srcDir := t.TempDir()
	destDir := t.TempDir()
	srcFileName := "test_dest_exists.xlsx"
	srcFilePath := filepath.Join(srcDir, srcFileName)
	existingDestPath := filepath.Join(destDir, srcFileName)

	// ソースファイル作成
	fSrc, err := os.Create(srcFilePath)
	if err != nil {
		t.Fatal(err)
	}
	fSrc.Close()
	// 移動先に同名ファイルを作成
	fDest, err := os.Create(existingDestPath)
	if err != nil {
		t.Fatal(err)
	}
	fDest.Close()

	// 2. Execute
	err = moveFileToSuccess(srcFilePath, destDir)

	// 3. Assert
	if err == nil {
		t.Fatal("移動先に同名ファイルが存在する場合にエラーが返されませんでした")
	}
	expectedErrMsg := fmt.Sprintf("移動先に同名ファイル '%s' が既に存在します", srcFileName)
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("期待されるエラーメッセージが含まれていません。\n期待含む: %s\n実際: %v", expectedErrMsg, err)
	}
	// 移動元ファイルが残っていることを確認
	if _, errStat := os.Stat(srcFilePath); os.IsNotExist(errStat) {
		t.Errorf("同名ファイルエラー時にソースファイルが削除されました: %s", srcFilePath)
	}
}

func TestMoveFileToSuccess_SrcFileNotExist(t *testing.T) {
	// 1. Setup
	srcDir := t.TempDir()
	destDir := t.TempDir() // 移動先ディレクトリは存在する
	srcFileName := "non_existent_src.xlsx"
	srcFilePath := filepath.Join(srcDir, srcFileName) // このファイルは作成しない

	// 2. Execute
	err := moveFileToSuccess(srcFilePath, destDir) // 存在しないファイルを指定

	// 3. Assert
	if err == nil {
		t.Fatal("移動元ファイルが存在しない場合にエラーが返されませんでした")
	}
	// エラーメッセージに "no such file or directory" などが含まれるか確認 (OS依存)
	if !strings.Contains(err.Error(), "no such file or directory") && // Linux/macOS
		!strings.Contains(err.Error(), "The system cannot find the file specified.") { // Windows
		t.Logf("移動元ファイルなしエラーのメッセージが想定と異なる可能性があります: %v", err)
	}
}
