# 30dayGameStats

Позволяет владельцам станций выгружать статистику по играм за последние 30 дней используя API сайта Drova.

Запуск

1. Скопируйте все файлы на свой локальный компьютер и распакуйте.
2. Установить Golang https://go.dev/
3. Запускаем copilate.bat, получаем исполняемый файл
4. Запускаем исполняемый файл, по окончании работы программа в папке появится файл csv.
5. Если возникнут ошибки, они будут записаны в файл errors.log.

Для корректного отображения кириллицы в Excel, файл с сессиями необходимо импортировать (Вкладка Данные->Из текстового/CSV-файла). Кодировка UTF-8, разделитель запятая