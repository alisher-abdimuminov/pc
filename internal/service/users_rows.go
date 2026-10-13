package service

import "pc/internal/db"

// listRowUser converts a joined list row back into a plain user model.
func listRowUser(r db.ListUsersRow) db.User {
	return db.User{
		ID: r.ID, Username: r.Username, PasswordHash: r.PasswordHash, Role: r.Role, IsActive: r.IsActive,
		MustChangePassword: r.MustChangePassword, LastLoginAt: r.LastLoginAt, HemisID: r.HemisID,
		FirstName: r.FirstName, SecondName: r.SecondName, ThirdName: r.ThirdName, FullName: r.FullName,
		ShortName: r.ShortName, Image: r.Image, HemisImgUrl: r.HemisImgUrl, BirthDate: r.BirthDate, Phone: r.Phone,
		Gender: r.Gender, PaymentForm: r.PaymentForm, Level: r.Level, Semester: r.Semester, Gpa: r.Gpa,
		Address: r.Address, Country: r.Country, Province: r.Province, District: r.District,
		GroupID: r.GroupID, FacultyID: r.FacultyID, FaceEnrolledAt: r.FaceEnrolledAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}
